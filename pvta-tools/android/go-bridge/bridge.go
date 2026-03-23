package pvtagoandroid

import core "github.com/mhlotto/vibrazioni/pvta-tools/pkg/mobilebridge"

type Bridge struct {
	inner *core.Bridge
}

func NewBridge() *Bridge {
	return &Bridge{inner: core.NewBridge()}
}

func (b *Bridge) SetBaseURL(baseURL string) {
	b.inner.SetBaseURL(baseURL)
}

func (b *Bridge) BaseURL() string {
	return b.inner.BaseURL()
}

func (b *Bridge) SetTimeoutSeconds(seconds int) {
	b.inner.SetTimeoutSeconds(seconds)
}

func (b *Bridge) RoutesJSON() (string, error) {
	return b.inner.RoutesJSON()
}

func (b *Bridge) VehiclesJSON() (string, error) {
	return b.inner.VehiclesJSON()
}

func (b *Bridge) StopsJSON(filter string) (string, error) {
	return b.inner.StopsJSON(filter)
}

func (b *Bridge) RouteStatusJSON(input string) (string, error) {
	return b.inner.RouteStatusJSON(input)
}

func (b *Bridge) StopStatusJSON(input string) (string, error) {
	return b.inner.StopStatusJSON(input)
}

func (b *Bridge) DeparturesJSON(input string) (string, error) {
	return b.inner.DeparturesJSON(input)
}

func (b *Bridge) RouteStopsJSON(input string) (string, error) {
	return b.inner.RouteStopsJSON(input)
}
