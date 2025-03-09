package nip47

import (
	"context"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
)

type WalletServiceInfo struct {
	EncryptionTypes   []string
	Capabilities      []string
	NotificationTypes []string
}

type GetInfoResult struct {
	Alias         string   `json:"alias"`
	Color         string   `json:"color"`
	Pubkey        string   `json:"pubkey"`
	Network       string   `json:"network"`
	BlockHeight   uint     `json:"block_height"`
	BlockHash     string   `json:"block_hash"`
	Methods       []string `json:"methods"`
	Notifications []string `json:"notifications"`
}

type MakeInvoiceParams struct {
	Amount          uint64      `json:"amount"`
	Expiry          *uint32     `json:"expiry"`
	Description     string      `json:"description"`
	DescriptionHash string      `json:"description_hash"`
	Metadata        interface{} `json:"metadata"`
}

type PayInvoiceParams struct {
	Invoice  string      `json:"invoice"`
	Amount   *uint64     `json:"amount"`
	Metadata interface{} `json:"metadata"`
}

type LookupInvoiceParams struct {
	PaymentHash string `json:"payment_hash"`
	Invoice     string `json:"invoice"`
}

type ListTransactionsParams struct {
	From           uint64 `json:"from"`
	To             uint64 `json:"to"`
	Limit          uint16 `json:"limit"`
	Offset         uint32 `json:"offset"`
	Unpaid         bool   `json:"unpaid"`
	UnpaidOutgoing bool   `json:"unpaid_outgoing"`
	UnpaidIncoming bool   `json:"unpaid_incoming"`
	Type           string `json:"type"`
}

type GetBalanceResult struct {
	Balance uint64 `json:"balance"`
}

type PayInvoiceResult struct {
	Preimage string `json:"preimage"`
	FeesPaid uint64 `json:"fees_paid"`
}

type MakeInvoiceResult = Transaction
type LookupInvoiceResult = Transaction
type ListTransactionsResult struct {
	Transactions []Transaction `json:"transactions"`
	TotalCount   uint32        `json:"total_count"`
}

type Transaction struct {
	Type            string      `json:"type"`
	State           string      `json:"state"`
	Invoice         string      `json:"invoice"`
	Description     string      `json:"description"`
	DescriptionHash string      `json:"description_hash"`
	Preimage        string      `json:"preimage"`
	PaymentHash     string      `json:"payment_hash"`
	Amount          uint64      `json:"amount"`
	FeesPaid        uint64      `json:"fees_paid"`
	CreatedAt       uint64      `json:"created_at"`
	ExpiresAt       uint64      `json:"expires_at"`
	SettledAt       *uint64     `json:"settled_at"`
	Metadata        interface{} `json:"metadata"`
}

type NWCClient struct {
	pool            *nostr.SimplePool
	relays          []string
	conversationKey [32]byte // nip44
	clientSecretKey string
	walletPublicKey string
}

var json = jsoniter.ConfigFastest

type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (err *ResponseError) Error() string {
	return fmt.Sprintf("%s %s", err.Code, err.Message)
}

type Response struct {
	ResultType string         `json:"result_type"`
	Error      *ResponseError `json:"error"`
	Result     interface{}    `json:"result"`
}

// creates a new NWC client from a NWC URI
func NewNWCClientFromURI(ctx context.Context, nwcUri string, pool *nostr.SimplePool) (client *NWCClient, err error) {
	nwcUriParts, err := ParseNWCURI(nwcUri)
	if err != nil {
		return nil, err
	}

	return NewNWCClient(ctx, nwcUriParts.clientSecretKey, nwcUriParts.walletPublicKey, nwcUriParts.relays, pool)
}

// creates a new NWC client from NWC URI parts
func NewNWCClient(ctx context.Context, clientSecretKey string, walletPublicKey string, relays []string, pool *nostr.SimplePool) (client *NWCClient, err error) {

	if pool == nil {
		pool = nostr.NewSimplePool(ctx)
	}

	conversationKey, err := nip44.GenerateConversationKey(walletPublicKey, clientSecretKey)
	if err != nil {
		return nil, err
	}

	return &NWCClient{
		pool:            pool,
		relays:          relays,
		clientSecretKey: clientSecretKey,
		conversationKey: conversationKey,
		walletPublicKey: walletPublicKey,
	}, nil
}

// fetches the NIP-47 info event (kind 13194)
func (client NWCClient) GetWalletServiceInfo(ctx context.Context) (*WalletServiceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	events := client.pool.SubscribeMany(ctx, client.relays, nostr.Filter{
		Limit:   1,
		Kinds:   []int{13194},
		Authors: []string{client.walletPublicKey}})

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled")
	case event := <-events:
		encryptionTypes := []string{}
		notificationTypes := []string{}
		encryptionTag := event.Tags.GetFirst([]string{"encryption"})
		notificationsTag := event.Tags.GetFirst([]string{"notifications"})
		if encryptionTag != nil {
			encryptionTypes = strings.Split((*encryptionTag).Value(), " ")
		}
		if notificationsTag != nil {
			notificationTypes = strings.Split((*notificationsTag).Value(), " ")
		}
		info := &WalletServiceInfo{
			EncryptionTypes:   encryptionTypes,
			NotificationTypes: notificationTypes,
			Capabilities:      strings.Split(event.Content, " "),
		}
		return info, nil
	}
}

// executes the NIP-47 get_info request method
func (client NWCClient) GetInfo(ctx context.Context) (*GetInfoResult, error) {
	getInfoResult := GetInfoResult{}
	err := client.RPC(ctx, "get_info", nil, &getInfoResult, nil)
	if err != nil {
		return nil, err
	}

	return &getInfoResult, nil
}

// executes the NIP-47 make_invoice request method
func (client NWCClient) MakeInvoice(ctx context.Context, params *MakeInvoiceParams) (*MakeInvoiceResult, error) {
	makeInvoiceResult := MakeInvoiceResult{}
	err := client.RPC(ctx, "make_invoice", params, &makeInvoiceResult, nil)
	if err != nil {
		return nil, err
	}

	return &makeInvoiceResult, nil
}

// executes the NIP-47 pay_invoice request method
func (client NWCClient) PayInvoice(ctx context.Context, params *PayInvoiceParams) (*PayInvoiceResult, error) {
	payInvoiceResult := PayInvoiceResult{}
	err := client.RPC(ctx, "pay_invoice", params, &payInvoiceResult, nil)
	if err != nil {
		return nil, err
	}

	return &payInvoiceResult, nil
}

// executes the NIP-47 lookup_invoice request method
func (client NWCClient) LookupInvoice(ctx context.Context, params *LookupInvoiceParams) (*LookupInvoiceResult, error) {
	lookupInvoiceResult := LookupInvoiceResult{}
	err := client.RPC(ctx, "lookup_invoice", params, &lookupInvoiceResult, nil)
	if err != nil {
		return nil, err
	}

	return &lookupInvoiceResult, nil
}

// executes the NIP-47 list_transactions request method
func (client NWCClient) ListTransactions(ctx context.Context, params *ListTransactionsParams) (*ListTransactionsResult, error) {
	listTransactionsResult := ListTransactionsResult{}
	err := client.RPC(ctx, "list_transactions", params, &listTransactionsResult, nil)
	if err != nil {
		return nil, err
	}

	return &listTransactionsResult, nil
}

// executes the NIP-47 get_balance request method
func (client NWCClient) GetBalance(ctx context.Context) (*GetBalanceResult, error) {
	getBalanceResult := GetBalanceResult{}
	err := client.RPC(ctx, "get_balance", nil, &getBalanceResult, nil)
	if err != nil {
		return nil, err
	}

	return &getBalanceResult, nil
}

type rpcOptions struct {
	timeoutSeconds *uint64
}

// executes a custom NIP-47 request method and waits for the response
func (client NWCClient) RPC(ctx context.Context, method string, params interface{}, result interface{}, opts *rpcOptions) error {
	timeoutSeconds := uint64(10)
	if opts != nil && opts.timeoutSeconds != nil {
		timeoutSeconds = *opts.timeoutSeconds
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	req, err := json.Marshal(Request{
		Method: method,
		Params: params,
	})
	if err != nil {
		return err
	}

	content, err := nip44.Encrypt(string(req), client.conversationKey)
	if err != nil {
		return fmt.Errorf("error encrypting request: %w", err)
	}

	evt := nostr.Event{
		Content:   content,
		CreatedAt: nostr.Now(),
		Kind:      23194,
		Tags:      nostr.Tags{{"p", client.walletPublicKey}, {"encryption", "nip44_v2"}},
	}
	if err := evt.Sign(client.clientSecretKey); err != nil {
		return fmt.Errorf("failed to sign request event: %w", err)
	}

	hasWorked := make(chan struct{})

	events := client.pool.SubscribeMany(ctx, client.relays, nostr.Filter{
		Limit:   1,
		Kinds:   []int{23195},
		Authors: []string{client.walletPublicKey},
		Tags:    nostr.TagMap{"e": []string{evt.ID}}})

	for _, url := range client.relays {
		go func(url string) {
			relay, err := client.pool.EnsureRelay(url)
			if err != nil {
				return
			}
			err = relay.Publish(ctx, evt)
			if err != nil {
				return
			}

			select {
			case hasWorked <- struct{}{}:
			default:
			}
		}(url)
	}

	select {
	case <-hasWorked:
		// continue
	case <-ctx.Done():
		return fmt.Errorf("couldn't connect to any relay")
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	case event := <-events:
		plain, err := nip44.Decrypt(event.Content, client.conversationKey)
		if err != nil {
			return err
		}

		resp := Response{
			Result: &result,
		}
		err = json.Unmarshal([]byte(plain), &resp)
		if err != nil {
			return err
		}

		return nil
	}
}
