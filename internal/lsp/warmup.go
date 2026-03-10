package lsp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	MaxWarmupRetries     = 3
	DefaultWarmupTimeout = 5 * time.Second
)

type ClientFactory func(language, rootPath string) (*Client, error)

var initializeClient = func(ctx context.Context, client *Client) error {
	return client.Initialize(ctx)
}

type WarmState struct {
	RequestedLanguages []string
	AttemptedLanguages []string
	Languages          map[string]LanguageWarmState
	Clients            map[string]*Client
}

type LanguageWarmState struct {
	ServerName    string
	Ready         bool
	RetryCount    int
	FailureReason string
}

func WarmLanguages(rootPath string, languages []string) (*WarmState, error) {
	return WarmLanguagesWithFactory(rootPath, languages, NewClient)
}

func WarmLanguagesWithFactory(rootPath string, languages []string, factory ClientFactory) (*WarmState, error) {
	requested := UniqueLSPKeys(languages)
	sort.Strings(requested)

	state := &WarmState{
		RequestedLanguages: requested,
		AttemptedLanguages: make([]string, 0, len(requested)),
		Languages:          make(map[string]LanguageWarmState, len(requested)),
		Clients:            make(map[string]*Client, len(requested)),
	}

	for _, language := range requested {
		server, ok := KnownLSPs[language]
		if !ok {
			state.Languages[language] = LanguageWarmState{
				ServerName:    language,
				FailureReason: "unsupported language",
			}
			continue
		}

		state.AttemptedLanguages = append(state.AttemptedLanguages, language)
		langState := LanguageWarmState{ServerName: server.Name}

		for attempt := 1; attempt <= MaxWarmupRetries; attempt++ {
			langState.RetryCount = attempt
			client, err := factory(language, rootPath)
			if err != nil {
				langState.FailureReason = err.Error()
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), DefaultWarmupTimeout)
			err = initializeClient(ctx, client)
			cancel()
			if err != nil {
				_ = client.Shutdown(context.Background())
				langState.FailureReason = err.Error()
				continue
			}

			langState.Ready = true
			langState.FailureReason = ""
			state.Languages[language] = langState
			state.Clients[language] = client
			goto nextLanguage
		}

		if langState.FailureReason == "" {
			langState.FailureReason = "warm-up failed"
		}
		state.Languages[language] = langState

	nextLanguage:
	}

	return state, nil
}

func (s *WarmState) ReadyLanguages() []string {
	if s == nil {
		return nil
	}
	ready := make([]string, 0, len(s.Clients))
	for language, state := range s.Languages {
		if state.Ready {
			ready = append(ready, language)
		}
	}
	sort.Strings(ready)
	return ready
}

func (s *WarmState) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var errs []error
	for language, client := range s.Clients {
		if client == nil {
			continue
		}
		if err := client.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", language, err))
		}
	}
	s.Clients = map[string]*Client{}
	return errors.Join(errs...)
}
