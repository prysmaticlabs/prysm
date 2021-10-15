package endtoend

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bazelbuild/rules_webtesting/go/webtest"
	"github.com/prysmaticlabs/prysm/testing/endtoend/components"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/tebeka/selenium"
)

var (
	host = "127.0.0.1"
)

func init() {
	if hIP, ok := os.LookupEnv("E2E_HOST_IP"); ok {
		host = hIP
	}
}

func TestValidatorUI(t *testing.T) {
	ctx := context.Background()
	if dl, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, dl)
		defer cancel()
	}

	wd, err := webtest.NewWebDriverSession(selenium.Capabilities{})
	require.NoError(t, err)
	ensureWebdriver(t, wd)

	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))
	e2eParams.TestParams.SetHost(host)

	v := components.NewValidatorNode(&e2etypes.E2EConfig{}, 0, 0, 0)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		require.NoError(t, helpers.WaitOnNodes(ctx, []e2etypes.ComponentRunner{v}, func() {
			wg.Done()
		}))
	}()
	wg.Wait()

	time.Sleep(5 * time.Second) // Wait 5 seconds for validator to start services.

	ensureValidator(t, wd)
	url, err := url.Parse(fmt.Sprintf("http://%s:%d", host, e2eParams.TestParams.ValidatorGatewayPort))
	require.NoError(t, err)
	require.NoError(t, wd.Get(url.String()))

	// Click active onboarding card button. This is the imported wallet setup.
	require.NoError(t, clickElement(wd, selenium.ByCSSSelector, "div.onboarding-card.active p.wallet-action > button"))

	// We should be on the "Imported Wallet Setup" page
	txt, err := elementText(wd, selenium.ByCSSSelector, "div.create-a-wallet div.text-white.text-3xl")
	require.NoError(t, err)
	require.Equal(t, txt, "Imported Wallet Setup")

	if err := wd.Quit(); err != nil {
		t.Logf("Error quitting webdriver: %v", err)
	}
}

// ensureWebdriver is running.
func ensureWebdriver(t *testing.T, wd selenium.WebDriver) {
	address, err := webtest.HTTPAddress()
	if err != nil {
		t.Fatal(err)
	}

	address = address.ResolveReference(&url.URL{Path: "/healthz"})

	require.NoError(t, wd.Get(address.String()))

	el, err := wd.FindElement(selenium.ByCSSSelector, "body")
	require.NoError(t, err)
	txt, err := el.Text()
	require.NoError(t, err)

	require.Equal(t, txt, "ok", "Webdriver is not OK")
}

func ensureValidator(t *testing.T, wd selenium.WebDriver) {
	address, err := url.Parse(fmt.Sprintf("http://%s:%d", host, e2eParams.TestParams.ValidatorMetricsPort))
	require.NoError(t, err)
	address = address.ResolveReference(&url.URL{Path: "/metrics"}) // TODO: Validator should have /healthz
	require.NoError(t, wd.Get(address.String()))

	el, err := wd.FindElement(selenium.ByCSSSelector, "body")
	require.NoError(t, err)
	txt, err := el.Text()
	require.NoError(t, err)

	if txt == "" {
		t.Errorf("Validator health check, empty response at %s", address.String())
	}
}

// find element and click on it
func clickElement(wd selenium.WebDriver, by, selector string) error {
	el, err := wd.FindElement(by, selector)
	if err != nil {
		return err
	}
	return el.Click()
}

func elementText(wd selenium.WebDriver, by, selector string) (string, error) {
	el, err := wd.FindElement(by, selector)
	if err != nil {
		return "", err
	}
	return el.Text()
}
