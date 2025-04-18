//go:build e2e

package e2e_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	clientchat "github.com/zestagio/chat-service/tests/e2e/client-chat"
	managerworkspace "github.com/zestagio/chat-service/tests/e2e/manager-workspace"
	wsstream "github.com/zestagio/chat-service/tests/e2e/ws-stream"
)

var _ = Describe("Manager Scheduling Smoke", Ordered, func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		clientChat        *clientchat.Chat
		clientStream      *wsstream.Stream
		clientStreamErrCh = make(chan error, 1)

		managerWs          *managerworkspace.Workspace
		managerStream      *wsstream.Stream
		managerStreamErrCh = make(chan error, 1)
	)

	BeforeAll(func() {
		ctx, cancel = context.WithCancel(suiteCtx)

		// Setup client.
		clientChat = newClientChat(ctx, clientsPool.Get())

		var err error
		clientStream, err = wsstream.New(wsstream.NewOptions(
			wsClientEndpoint,
			wsClientOrigin,
			wsClientSecProtocol,
			clientChat.AccessToken(),
			clientChat.HandleEvent,
		))
		Expect(err).ShouldNot(HaveOccurred())
		go func() { clientStreamErrCh <- clientStream.Run(ctx) }()

		// Setup manager.
		managerWs = newManagerWs(ctx, managersPool.Get())

		managerStream, err = wsstream.New(wsstream.NewOptions(
			wsManagerEndpoint,
			wsManagerOrigin,
			wsManagerSecProtocol,
			managerWs.AccessToken(),
			managerWs.HandleEvent,
		))
		Expect(err).ShouldNot(HaveOccurred())
		go func() { managerStreamErrCh <- managerStream.Run(ctx) }()
	})

	AfterAll(func() {
		cancel()
		Expect(<-clientStreamErrCh).ShouldNot(HaveOccurred())
		Expect(<-managerStreamErrCh).ShouldNot(HaveOccurred())
	})

	It("no chats at the start of working day", func() {
		err := managerWs.Refresh(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		n := managerWs.ChatsCount()
		Expect(n).Should(Equal(0))
	})

	It("manager assigned to new problem", func() {
		err := managerWs.ReadyToNewProblems(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		err = clientChat.SendMessage(ctx, "Hello, sir!")
		Expect(err).ShouldNot(HaveOccurred())

		// Client side.

		waitForEvent(clientStream) // NewMessageEvent.
		waitForEvent(clientStream) // MessageSentEvent.
		waitForEvent(clientStream) // NewMessageEvent (service).

		msg, ok := clientChat.LastMessage()
		Expect(ok).Should(BeTrue())
		Expect(msg.Body).Should(Equal(fmt.Sprintf("Manager %s will answer you", managerWs.ManagerID())))

		// Manager side.

		waitForEvent(managerStream)         // NewChatEvent.
		waitForOptionalEvent(managerStream) // NewMessageEvent.

		n := managerWs.ChatsCount()
		Expect(n).Should(Equal(1))

		newChat, ok := managerWs.LastChat()
		Expect(ok).Should(BeTrue())
		Expect(newChat.ClientID.String()).Should(Equal(clientChat.ClientID().String()))
		Expect(newChat.ID).ShouldNot(BeEmpty())
	})

	It("assigned problem does not disappear", func() {
		err := managerWs.Refresh(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		n := managerWs.ChatsCount()
		Expect(n).Should(Equal(1))
	})

	It("manager see chat history", func() {
		lastChat, ok := managerWs.LastChat()
		Expect(ok).Should(BeTrue())

		n := lastChat.MessagesCount()
		Expect(n).Should(Equal(1))

		lastMsg, ok := lastChat.LastMessage()
		Expect(ok).Should(BeTrue())
		Expect(lastMsg.ID).ShouldNot(BeEmpty())
		Expect(lastMsg.ChatID).Should(Equal(lastChat.ID))
		Expect(lastMsg.AuthorID.String()).Should(Equal(clientChat.ClientID().String()))
		Expect(lastMsg.CreatedAt.IsZero()).Should(BeFalse())
	})

	It("manager answers back", func() {
		const response = "Hello, how can I help?"
		lastChat, _ := managerWs.LastChat()
		err := managerWs.SendMessage(ctx, lastChat.ID, response)
		Expect(err).ShouldNot(HaveOccurred())

		// Client side.

		waitForEvent(clientStream) // NewMessageEvent.

		msg, ok := clientChat.LastMessage()
		Expect(ok).Should(BeTrue())
		Expect(msg.ID).ShouldNot(BeEmpty())
		Expect(msg.AuthorID.String()).Should(Equal(managerWs.ManagerID().String()))
		Expect(msg.Body).Should(Equal(response))
		Expect(msg.IsService).Should(BeFalse())
		Expect(msg.CreatedAt.IsZero()).Should(BeFalse())

		// Manager side.

		waitForEvent(managerStream) // NewMessageEvent.

		lastChat, _ = managerWs.LastChat()
		mngrMsg, ok := lastChat.LastMessage()
		Expect(ok).Should(BeTrue())
		Expect(mngrMsg.ID).ShouldNot(BeEmpty())
		Expect(mngrMsg.ChatID).Should(Equal(lastChat.ID))
		Expect(mngrMsg.AuthorID.String()).Should(Equal(managerWs.ManagerID().String()))
		Expect(mngrMsg.Body).Should(Equal(response))
		Expect(mngrMsg.CreatedAt.IsZero()).Should(BeFalse())

		err = managerWs.Refresh(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		n := lastChat.MessagesCount()
		Expect(n).Should(Equal(2))
	})

	It("manager closes chat", func() {
		chatsNum := managerWs.ChatsCount()
		lastChat, _ := managerWs.LastChat()

		err := managerWs.CloseChat(ctx, lastChat.ID)
		Expect(err).ShouldNot(HaveOccurred())

		// Manager side.

		waitForEvent(managerStream) // ChatClosedEvent.

		n := managerWs.ChatsCount()
		Expect(n).Should(Equal(chatsNum - 1))

		canTakeMore := managerWs.CanTakeMoreProblems()
		Expect(canTakeMore).Should(BeTrue())

		// Client side.

		waitForEvent(clientStream) // NewMessageEvent (service).

		msg, ok := clientChat.LastMessage()
		Expect(ok).Should(BeTrue())
		Expect(msg.ID).ShouldNot(BeEmpty())
		Expect(msg.AuthorID.IsZero()).Should(BeTrue())
		Expect(msg.Body).Should(Equal(`Your question has been marked as resolved.
Thank you for being with us!`))
		Expect(msg.IsService).Should(BeTrue())
		Expect(msg.CreatedAt.IsZero()).Should(BeFalse())

		err = clientChat.Refresh(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		/*
			- "Hello, sir!"
			- "Manager %s will answer you"
			- "Hello, how can I help?"
			- "Your question has been marked as resolved..."
		*/
		Expect(clientChat.MessagesCount()).Should(Equal(4))
	})
})
