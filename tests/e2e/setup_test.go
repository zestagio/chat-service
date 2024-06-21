//go:build e2e

package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/types"
	apiclientv1 "github.com/zestagio/chat-service/tests/e2e/api/client/v1"
	clientchat "github.com/zestagio/chat-service/tests/e2e/client-chat"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E Suite")
}

var (
	suiteCtx       context.Context
	suiteCtxCancel context.CancelFunc

	kc *keycloakclient.Client

	apiClientV1Endpoint string
	clientsPool         *usersPool
)

var _ = ginkgo.BeforeSuite(func() {
	suiteCtx, suiteCtxCancel = context.WithCancel(context.Background())
	ginkgo.DeferCleanup(suiteCtxCancel)

	apiClientV1Endpoint = expectEnv("E2E_CLIENT_V1_API_ENDPOINT")

	kcBasePath := expectEnv("E2E_KEYCLOAK_BASE_PATH")
	kcRealm := expectEnv("E2E_KEYCLOAK_REALM")
	kcClientID := expectEnv("E2E_KEYCLOAK_CLIENT_ID")
	kcClientSecret := expectEnv("E2E_KEYCLOAK_CLIENT_SECRET")
	kcClientDebug, _ := strconv.ParseBool(expectEnv("E2E_KEYCLOAK_CLIENT_DEBUG"))
	kcClients := expectEnv("E2E_KEYCLOAK_CLIENTS") // "client1,client2,client3"

	var err error
	kc, err = keycloakclient.New(keycloakclient.NewOptions(
		kcBasePath,
		kcRealm,
		kcClientID,
		kcClientSecret,
		keycloakclient.WithDebugMode(kcClientDebug),
	))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	clients, err := parseUsers(kcClients)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	ginkgo.GinkgoWriter.Printf("clients: %v", clients)
	clientsPool = newUsersPool(clients)
})

func expectEnv(k string) string {
	v := os.Getenv(k)
	gomega.Expect(v).NotTo(gomega.BeZero(), fmt.Sprintf("Please make sure %q is set correctly.", k))
	return v
}

func parseUsers(s string) ([]user, error) {
	userNames := strings.Split(s, ",")
	if len(userNames) == 0 {
		return nil, errors.New("no users specified")
	}

	known := make(map[string]struct{}, len(userNames))
	result := make([]user, 0, len(userNames))
	for _, uname := range userNames {
		if _, ok := known[uname]; ok {
			return nil, fmt.Errorf("duplicated user: %v", uname)
		}

		result = append(result, user{
			Name:     uname,
			Password: uname, // NOTE: E2E client username & password must be equal.
		})
		known[uname] = struct{}{}
	}
	return result, nil
}

func newClientChat(ctx context.Context, client user) *clientchat.Chat {
	apiClientV1, token := newClientAPI(ctx, client)

	var cl simpleClaims
	t, _, err := new(jwt.Parser).ParseUnverified(token, &cl)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	clientID, err := types.Parse[types.UserID](t.Claims.(*simpleClaims).Subject)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	ginkgo.GinkgoWriter.Printf("client %v has token sub %v\n", client.Name, clientID)

	clientChat, err := clientchat.New(clientchat.NewOptions(clientID, token, apiClientV1))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	return clientChat
}

func newClientAPI(ctx context.Context, client user) (*apiclientv1.ClientWithResponses, string) {
	token, err := kc.Auth(ctx, client.Name, client.Password)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	authorizator := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		return nil
	}
	apiClientV1, err := apiclientv1.NewClientWithResponses(
		apiClientV1Endpoint,
		apiclientv1.WithRequestEditorFn(authorizator),
	)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	return apiClientV1, token.AccessToken
}

type user struct {
	Name     string
	Password string
}

type usersPool struct {
	users []user
	mu    sync.Mutex
}

func newUsersPool(users []user) *usersPool {
	p := &usersPool{
		users: make([]user, len(users)),
		mu:    sync.Mutex{},
	}
	copy(p.users, users)
	return p
}

func (p *usersPool) Get() user {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.users) == 0 {
		ginkgo.AbortSuite("there are no users in the pool - let's add a new one to Keycloak")
	}

	u := p.users[0]
	p.users = p.users[1:]

	ginkgo.GinkgoWriter.Printf("user %s removed from pool\n", u.Name)
	return u
}

type simpleClaims struct {
	Subject string `json:"sub,omitempty"`
}

func (sc simpleClaims) Valid() error {
	return nil
}
