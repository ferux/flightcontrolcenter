package telegram

import "context"

type Mock struct{}

func (Mock) SendMessageViaHTTP(_ context.Context, _, _, _ string) error { return nil }
