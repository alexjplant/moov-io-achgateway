// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/sethvargo/go-retry"
)

// MultiSender is a Sender which will attempt to send each Message to every
// included Sender and returns the first error encountered.
type MultiSender struct {
	logger      log.Logger
	senders     []Sender
	retryConfig *service.NotificationRetries
}

func NewMultiSender(logger log.Logger, cfg *service.Notifications, notifiers *service.UploadNotifiers) (*MultiSender, error) {
	ms := &MultiSender{logger: logger}
	if cfg == nil {
		return ms, nil
	}
	if cfg.Retry != nil {
		ms.retryConfig = cfg.Retry
	}

	emails := cfg.FindEmails(notifiers.Email)
	for i := range emails {
		sender, err := NewEmail(&emails[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	pds := cfg.FindPagerDutys(notifiers.PagerDuty)
	for i := range pds {
		sender, err := NewPagerDuty(&pds[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	slacks := cfg.FindSlacks(notifiers.Slack)
	for i := range slacks {
		sender, err := NewSlack(&slacks[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	ms.logger.Logf("multi-sender: created senders for %v", strings.Join(ms.senderTypes(), ", "))
	return ms, nil
}

func setupBackoff(cfg *service.NotificationRetries) (retry.Backoff, error) {
	fib, err := retry.NewFibonacci(cfg.Interval)
	if err != nil {
		return nil, fmt.Errorf("problem creating fibonacci: %v", err)
	}
	fib = retry.WithMaxRetries(cfg.MaxRetries, fib)
	return fib, nil
}

func (ms *MultiSender) senderTypes() []string {
	var out []string
	for i := range ms.senders {
		out = append(out, fmt.Sprintf("%T", ms.senders[i]))
	}
	return out
}

func (ms *MultiSender) Info(msg *Message) error {
	var firstError error
	for i := range ms.senders {
		err := ms.retry(func() error {
			return ms.senders[i].Info(msg)
		})
		if err != nil {
			ms.logger.Logf("multi-sender: Info %T: %v", ms.senders[i], err)
			if firstError == nil {
				firstError = err
			}
		}
	}
	return firstError
}

func (ms *MultiSender) Critical(msg *Message) error {
	var firstError error
	for i := range ms.senders {
		err := ms.retry(func() error {
			return ms.senders[i].Critical(msg)
		})
		if err != nil {
			ms.logger.Logf("multi-sender: Critical %T: %v", ms.senders[i], err)
			if firstError == nil {
				firstError = err
			}
		}
	}
	return firstError
}

func (ms *MultiSender) retry(f func() error) error {
	if ms.retryConfig != nil {
		backoff, err := setupBackoff(ms.retryConfig)
		if err != nil {
			return fmt.Errorf("retry: %v", err)
		}
		ctx := context.Background()
		return retry.Do(ctx, backoff, func(ctx context.Context) error {
			return isRetryableError(f())
		})
	}
	return f()
}

func isRetryableError(err error) error {
	if err == nil {
		return nil
	}
	if os.IsTimeout(err) || strings.Contains(err.Error(), "no such host") {
		return retry.RetryableError(err)
	}
	return nil
}
