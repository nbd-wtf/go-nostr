package nip47

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWalletServiceInfo(t *testing.T) {
	client := createTestClient(t)

	walletServiceInfo, err := client.GetWalletServiceInfo(context.TODO())
	require.NoError(t, err)
	require.NotNil(t, walletServiceInfo)
	assert.Contains(t, walletServiceInfo.Capabilities, "get_info")
	assert.Contains(t, walletServiceInfo.NotificationTypes, "payment_received")
	assert.Contains(t, walletServiceInfo.EncryptionTypes, "nip44_v2")
}

func TestGetInfo(t *testing.T) {
	client := createTestClient(t)

	getInfoResult, err := client.GetInfo(context.TODO())
	require.NoError(t, err)
	require.NotNil(t, getInfoResult)
	assert.Contains(t, getInfoResult.Methods, "get_info")
	assert.Contains(t, getInfoResult.Notifications, "payment_received")
	assert.Greater(t, getInfoResult.BlockHeight, uint(840_000))
	assert.Equal(t, 64, len(getInfoResult.BlockHash))
	assert.Equal(t, "mainnet", getInfoResult.Network)
}

func TestMakeInvoice(t *testing.T) {
	client := createTestClient(t)

	makeInvoiceResult, err := client.MakeInvoice(context.TODO(), &MakeInvoiceParams{
		Amount: uint64(1000),
	})
	require.NoError(t, err)
	require.NotNil(t, makeInvoiceResult)
	assert.Equal(t, makeInvoiceResult.Amount, uint64(1000))
	assert.True(t, strings.HasPrefix(makeInvoiceResult.Invoice, "lnbc"))
	assert.Equal(t, "pending", makeInvoiceResult.State)
	assert.Nil(t, makeInvoiceResult.SettledAt)
	assert.Greater(t, makeInvoiceResult.ExpiresAt, uint64(time.Now().Unix()))
	assert.Empty(t, makeInvoiceResult.Preimage)
}

func TestLookupInvoice(t *testing.T) {
	client := createTestClient(t)

	makeInvoiceResult, err := client.MakeInvoice(context.TODO(), &MakeInvoiceParams{
		Amount: uint64(1000),
	})

	require.NoError(t, err)
	require.NotNil(t, makeInvoiceResult)

	lookupInvoiceResult, err := client.LookupInvoice(context.TODO(), &LookupInvoiceParams{
		PaymentHash: makeInvoiceResult.PaymentHash,
	})

	require.NoError(t, err)
	require.NotNil(t, lookupInvoiceResult)

	require.NoError(t, err)
	require.NotNil(t, lookupInvoiceResult)
	assert.Equal(t, lookupInvoiceResult.Amount, uint64(1000))
	assert.True(t, strings.HasPrefix(lookupInvoiceResult.Invoice, "lnbc"))
	assert.Equal(t, "pending", lookupInvoiceResult.State)
	assert.Nil(t, lookupInvoiceResult.SettledAt)
	assert.Greater(t, lookupInvoiceResult.ExpiresAt, uint64(time.Now().Unix()))
	assert.Empty(t, lookupInvoiceResult.Preimage)
}

func TestListTransactions(t *testing.T) {
	client := createTestClient(t)

	makeInvoiceResult, err := client.MakeInvoice(context.TODO(), &MakeInvoiceParams{
		Amount: uint64(1000),
	})

	require.NoError(t, err)
	require.NotNil(t, makeInvoiceResult)

	listTransactionsResult, err := client.ListTransactions(context.TODO(), &ListTransactionsParams{
		Unpaid: true,
	})

	require.NoError(t, err)
	require.NotNil(t, listTransactionsResult)
	require.NotZero(t, len(listTransactionsResult.Transactions))
	require.NotZero(t, listTransactionsResult.TotalCount)

	transaction := listTransactionsResult.Transactions[0]

	require.NoError(t, err)
	require.NotNil(t, transaction)
	assert.Equal(t, transaction.Amount, uint64(1000))
	assert.True(t, strings.HasPrefix(transaction.Invoice, "lnbc"))
	assert.Equal(t, "pending", transaction.State)
	assert.Nil(t, transaction.SettledAt)
	assert.Greater(t, transaction.ExpiresAt, uint64(time.Now().Unix()))
	assert.Empty(t, transaction.Preimage)
}
func TestGetBalance(t *testing.T) {
	client := createTestClient(t)

	getBalanceResult, err := client.GetBalance(context.TODO())

	require.NoError(t, err)
	require.NotNil(t, getBalanceResult)

	assert.Equal(t, uint64(100_000), getBalanceResult.Balance)
}

func TestPayInvoice(t *testing.T) {
	client := createTestClient(t)

	makeInvoiceResult, err := client.MakeInvoice(context.TODO(), &MakeInvoiceParams{
		Amount: uint64(1000),
	})

	require.NoError(t, err)
	require.NotNil(t, makeInvoiceResult)

	payInvoiceResult, err := client.PayInvoice(context.TODO(), &PayInvoiceParams{
		Invoice: makeInvoiceResult.Invoice,
	})

	require.NoError(t, err)
	require.NotNil(t, payInvoiceResult)
	assert.Equal(t, 64, len(payInvoiceResult.Preimage))
	assert.Equal(t, uint64(0), payInvoiceResult.FeesPaid)

	require.NoError(t, err)
	require.NotNil(t, makeInvoiceResult)

	lookupInvoiceResult, err := client.LookupInvoice(context.TODO(), &LookupInvoiceParams{
		PaymentHash: makeInvoiceResult.PaymentHash,
	})

	require.NoError(t, err)
	require.NotNil(t, lookupInvoiceResult)

	require.NoError(t, err)
	require.NotNil(t, lookupInvoiceResult)
	assert.Equal(t, lookupInvoiceResult.Amount, uint64(1000))
	assert.True(t, strings.HasPrefix(lookupInvoiceResult.Invoice, "lnbc"))
	assert.Equal(t, "settled", lookupInvoiceResult.State)
	require.NotNil(t, lookupInvoiceResult.SettledAt)
	assert.LessOrEqual(t, *lookupInvoiceResult.SettledAt, uint64(time.Now().Unix()))
	assert.Greater(t, lookupInvoiceResult.ExpiresAt, uint64(time.Now().Unix()))
	assert.Equal(t, 64, len(lookupInvoiceResult.Preimage))
}

func createTestClient(t *testing.T) *NWCClient {
	nwcUri := os.Getenv("NWC_URI")
	if nwcUri == "" {
		t.Skip()
		return nil
	}
	client, err := NewNWCClientFromURI(context.TODO(), nwcUri, nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	return client
}
