package metrics

import (
	"context"

	"github.com/navyansh007/kharcha-bot/internal/telegram"
)

type countingClient struct {
	telegram.Client
	metrics *Metrics
}

// Instrument returns a client that counts every send made through client.
func (m *Metrics) Instrument(client telegram.Client) telegram.Client {
	return countingClient{Client: client, metrics: m}
}

func (c countingClient) SendMessage(ctx context.Context, params telegram.SendMessageParams) error {
	err := c.Client.SendMessage(ctx, params)
	c.metrics.ObserveSend(err)
	return err
}

func (c countingClient) EditMessageText(ctx context.Context, params telegram.EditMessageTextParams) error {
	err := c.Client.EditMessageText(ctx, params)
	c.metrics.ObserveSend(err)
	return err
}

func (c countingClient) SendDocument(ctx context.Context, doc telegram.Document) error {
	err := c.Client.SendDocument(ctx, doc)
	c.metrics.ObserveSend(err)
	return err
}
