package server

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"golang.org/x/net/context"

	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/client/enterprise"
	"github.com/pachyderm/pachyderm/src/client/pkg/require"
	"github.com/pachyderm/pachyderm/src/server/pkg/backoff"
	"github.com/pachyderm/pachyderm/src/server/pkg/testutil"
)

var (
	pachClient *client.APIClient
	clientOnce sync.Once
)

const year = 365 * 24 * time.Hour

// getPachClient creates a seed client with a grpc connection to a pachyderm
// cluster
func getPachClient(t testing.TB) *client.APIClient {
	clientOnce.Do(func() {
		var err error
		if _, ok := os.LookupEnv("PACHD_PORT_650_TCP_ADDR"); ok {
			pachClient, err = client.NewInCluster()
		} else {
			pachClient, err = client.NewForTest()
		}
		if err != nil {
			t.Fatalf("error getting Pachyderm client: %s", err.Error())
		}
	})
	return pachClient
}

func TestValidateActivationCode(t *testing.T) {
	_, err := validateActivationCode(testutil.GetTestEnterpriseCode())
	require.NoError(t, err)
}

func TestGetState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	client := getPachClient(t)

	// Activate Pachyderm Enterprise and make sure the state is ACTIVE
	_, err := client.Enterprise.Activate(context.Background(),
		&enterprise.ActivateRequest{ActivationCode: testutil.GetTestEnterpriseCode()})
	require.NoError(t, err)
	require.NoError(t, backoff.Retry(func() error {
		resp, err := client.Enterprise.GetState(context.Background(),
			&enterprise.GetStateRequest{})
		if err != nil {
			return err
		}
		if resp.State != enterprise.State_ACTIVE {
			return fmt.Errorf("expected enterprise state to be ACTIVE but was %v", resp.State)
		}
		expires, err := types.TimestampFromProto(resp.Info.Expires)
		if err != nil {
			return err
		}
		if time.Until(expires) <= year {
			return fmt.Errorf("expected test token to expire >1yr in the future, but expires at %v (congratulations on making it to 2026!)", expires)
		}
		if resp.ActivationCode != testutil.GetTestEnterpriseCode() {
			return fmt.Errorf("incorrect activation code, got: %s, expected: %s", resp.ActivationCode, testutil.GetTestEnterpriseCode())
		}
		return nil
	}, backoff.NewTestingBackOff()))

	// Make current enterprise token expire
	expires := time.Now().Add(-30 * time.Second)
	expiresProto, err := types.TimestampProto(expires)
	require.NoError(t, err)
	_, err = client.Enterprise.Activate(context.Background(),
		&enterprise.ActivateRequest{
			ActivationCode: testutil.GetTestEnterpriseCode(),
			Expires:        expiresProto,
		})
	require.NoError(t, err)
	require.NoError(t, backoff.Retry(func() error {
		resp, err := client.Enterprise.GetState(context.Background(),
			&enterprise.GetStateRequest{})
		if err != nil {
			return err
		}
		if resp.State != enterprise.State_EXPIRED {
			return fmt.Errorf("expected enterprise state to be EXPIRED but was %v", resp.State)
		}
		respExpires, err := types.TimestampFromProto(resp.Info.Expires)
		if err != nil {
			return err
		}
		if expires.Unix() != respExpires.Unix() {
			return fmt.Errorf("expected enterprise expiration to be %v, but was %v", expires, respExpires)
		}
		if resp.ActivationCode != testutil.GetTestEnterpriseCode() {
			return fmt.Errorf("incorrect activation code, got: %s, expected: %s", resp.ActivationCode, testutil.GetTestEnterpriseCode())
		}
		return nil
	}, backoff.NewTestingBackOff()))
}

func TestDeactivate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	client := getPachClient(t)

	// Activate Pachyderm Enterprise and make sure the state is ACTIVE
	_, err := client.Enterprise.Activate(context.Background(),
		&enterprise.ActivateRequest{ActivationCode: testutil.GetTestEnterpriseCode()})
	require.NoError(t, err)
	require.NoError(t, backoff.Retry(func() error {
		resp, err := client.Enterprise.GetState(context.Background(),
			&enterprise.GetStateRequest{})
		if err != nil {
			return err
		}
		if resp.State != enterprise.State_ACTIVE {
			return fmt.Errorf("expected enterprise state to be ACTIVE but was %v", resp.State)
		}
		return nil
	}, backoff.NewTestingBackOff()))

	// Deactivate cluster and make sure its state is NONE
	_, err = client.Enterprise.Deactivate(context.Background(),
		&enterprise.DeactivateRequest{})
	require.NoError(t, err)
	require.NoError(t, backoff.Retry(func() error {
		resp, err := client.Enterprise.GetState(context.Background(),
			&enterprise.GetStateRequest{})
		if err != nil {
			return err
		}
		if resp.State != enterprise.State_NONE {
			return fmt.Errorf("expected enterprise state to be NONE but was %v", resp.State)
		}
		return nil
	}, backoff.NewTestingBackOff()))
}

// TestDoubleDeactivate makes sure calling Deactivate() when there is no
// enterprise token works. Fixes
// https://github.com/pachyderm/pachyderm/issues/3013
func TestDoubleDeactivate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	client := getPachClient(t)

	// Deactivate cluster and make sure its state is NONE (enterprise might be
	// active at the start of this test?)
	_, err := client.Enterprise.Deactivate(context.Background(),
		&enterprise.DeactivateRequest{})
	require.NoError(t, err)
	require.NoError(t, backoff.Retry(func() error {
		resp, err := client.Enterprise.GetState(context.Background(),
			&enterprise.GetStateRequest{})
		if err != nil {
			return err
		}
		if resp.State != enterprise.State_NONE {
			return fmt.Errorf("expected enterprise state to be NONE but was %v", resp.State)
		}
		return nil
	}, backoff.NewTestingBackOff()))

	// Deactivate the cluster again to make sure deactivation with no token works
	_, err = client.Enterprise.Deactivate(context.Background(),
		&enterprise.DeactivateRequest{})
	require.NoError(t, err)
	resp, err := client.Enterprise.GetState(context.Background(),
		&enterprise.GetStateRequest{})
	require.NoError(t, err)
	require.Equal(t, enterprise.State_NONE, resp.State)
}
