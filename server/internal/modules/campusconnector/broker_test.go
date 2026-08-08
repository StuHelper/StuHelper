package campusconnector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBrokerIsProcessLocalSingleDeliveryAndWipesOwnedPassword(t *testing.T) {
	broker := NewBroker(2)
	t.Cleanup(broker.Close)
	inputPassword := []byte("user-password")
	resultChannel := make(chan InteractiveResult, 1)
	errorChannel := make(chan error, 1)
	go func() {
		result, err := broker.Submit(context.Background(), InteractiveRequest{
			ID: "request-1", NodeID: "node-1", SchoolID: 1, SchoolCode: "0000000001",
			OperationKey: "school.account.authenticate", AdapterID: "school_ldap_bind",
			AdapterVersion: "1", StudentID: "20990001", Password: inputPassword,
			DeadlineAt: time.Now().Add(time.Minute),
		})
		resultChannel <- result
		errorChannel <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	delivery, err := broker.Claim(ctx, "node-1")
	require.NoError(t, err)
	require.Equal(t, inputPassword, delivery.Password)
	require.NotSame(t, &inputPassword[0], &delivery.Password[0], "broker must own a bounded copy")
	require.ErrorIs(t, broker.Complete("node-2", "request-1", InteractiveResult{}), ErrRequestNotFound)
	require.NoError(t, broker.Complete("node-1", "request-1", InteractiveResult{
		ResultCode: ResultSuccess, AccountSubject: "stable-subject", StudentID: "20990001",
	}))
	require.NoError(t, <-errorChannel)
	require.Equal(t, ResultSuccess, (<-resultChannel).ResultCode)
	require.Eventually(t, func() bool {
		for _, value := range delivery.Password {
			if value != 0 {
				return false
			}
		}
		return true
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, []byte("user-password"), inputPassword, "caller-owned request buffer is not mutated")
	require.ErrorIs(t, broker.Complete("node-1", "request-1", InteractiveResult{}), ErrRequestNotFound)
}
