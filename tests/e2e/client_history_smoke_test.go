//go:build e2e

package e2e_test

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/zestagio/chat-service/internal/types"
	clientchat "github.com/zestagio/chat-service/tests/e2e/client-chat"
)

var _ = ginkgo.Describe("Client History Smoke", ginkgo.Ordered, func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		clientChat *clientchat.Chat
	)

	ginkgo.BeforeAll(func() {
		ctx, cancel = context.WithCancel(suiteCtx)

		// Setup client.
		clientChat = newClientChat(ctx, clientsPool.Get())
	})

	ginkgo.AfterAll(func() {
		cancel()
	})

	ginkgo.It("no chat messages at the start of communication", func() {
		err := clientChat.Refresh(ctx)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		n := clientChat.MessagesCount()
		gomega.Expect(n).Should(gomega.Equal(0))
	})

	ginkgo.It("client send the 1st message without errors", func() {
		err := clientChat.SendMessage(ctx, "Hello!")
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		msg, ok := clientChat.LastMessage()
		gomega.Expect(ok).Should(gomega.BeTrue())
		gomega.Expect(msg.AuthorID).Should(gomega.Equal(clientChat.ClientID()))
	})

	secondMsgReqID := types.NewRequestID()

	ginkgo.It("client send the 2nd message without errors", func() {
		err := clientChat.SendMessage(ctx, "I have a problem :(", clientchat.WithRequestID(secondMsgReqID))
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		msg, ok := clientChat.LastMessage()
		gomega.Expect(ok).Should(gomega.BeTrue())
		gomega.Expect(msg.AuthorID).Should(gomega.Equal(clientChat.ClientID()))
	})

	ginkgo.It("client retries the 2nd message without errors", func() {
		err := clientChat.SendMessage(ctx, "I have a problem :(", clientchat.WithRequestID(secondMsgReqID))
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		msg, ok := clientChat.LastMessage()
		gomega.Expect(ok).Should(gomega.BeTrue())
		gomega.Expect(msg.AuthorID).Should(gomega.Equal(clientChat.ClientID()))
	})

	ginkgo.It("we still have two messages in the chat", func() {
		n := clientChat.MessagesCount()
		gomega.Expect(n).Should(gomega.Equal(2))
	})

	ginkgo.It("we have two messages in the history", func() {
		sentMessages := clientChat.Messages()

		err := clientChat.Refresh(ctx)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		for i := 0; i < 3; i++ {
			err := clientChat.GetHistory(ctx)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}
		history := clientChat.Messages()
		gomega.Expect(sentMessages).Should(gomega.Equal(history))

		for _, m := range history {
			gomega.Expect(m.AuthorID).Should(gomega.Equal(clientChat.ClientID()))
		}
	})
})
