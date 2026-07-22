package client

import (
	"testing"
)

func TestApplyGitHubMirror_EnvOverride(t *testing.T) {
	const ghURL = "https://github.com/SagerNet/sing-box/releases/download/v1.0/x.tar.gz"

	t.Run("mirror off", func(t *testing.T) {
		t.Setenv(mirrorEnvKey, "off")
		if got := applyGitHubMirror(ghURL); got != ghURL {
			t.Errorf("should not add mirror when off, got %s", got)
		}
	})

	t.Run("custom mirror prefix", func(t *testing.T) {
		t.Setenv(mirrorEnvKey, "https://my.mirror/")
		want := "https://my.mirror/" + ghURL
		if got := applyGitHubMirror(ghURL); got != want {
			t.Errorf("custom mirror mismatch, expected %s, got %s", want, got)
		}
	})
}

// Non-github.com URLs (e.g. api.github.com) must not be rewritten.
func TestApplyGitHubMirror_NonGitHubUntouched(t *testing.T) {
	t.Setenv(mirrorEnvKey, "https://my.mirror/")
	apiURL := "https://api.github.com/repos/SagerNet/sing-box/releases/latest"
	if got := applyGitHubMirror(apiURL); got != apiURL {
		t.Errorf("non-github.com URL must not be rewritten, got %s", got)
	}
}

// TestGeoProviderParsers validates each geo endpoint's country-code parsing
// (pure function, no network).
func TestGeoProviderParsers(t *testing.T) {
	byName := map[string]geoProvider{}
	for _, p := range geoProviders {
		byName[p.name] = p
	}

	// cloudflare trace: multi-line key=value, take loc=
	cfBody := "fl=963f63\nh=www.cloudflare.com\nip=1.2.3.4\nloc=CN\ntls=TLSv1.3\n"
	if got := byName["cloudflare"].parse(cfBody); got != "CN" {
		t.Errorf("cloudflare should parse CN, got %q", got)
	}
	cfBodySG := "colo=SIN\nloc=SG\nwarp=off\n"
	if got := byName["cloudflare"].parse(cfBodySG); got != "SG" {
		t.Errorf("cloudflare should parse SG, got %q", got)
	}
	if got := byName["cloudflare"].parse("no loc here\n"); got != "" {
		t.Errorf("no loc line should return empty, got %q", got)
	}

	// ip.sb JSON
	if got := byName["ip.sb"].parse(`{"country_code":"CN","ip":"1.2.3.4"}`); got != "CN" {
		t.Errorf("ip.sb should parse CN, got %q", got)
	}
	if got := byName["ip.sb"].parse(`not json`); got != "" {
		t.Errorf("invalid JSON should return empty, got %q", got)
	}

	// ip-api plain text
	if got := byName["ip-api"].parse("CN\n"); got != "CN" {
		t.Errorf("ip-api should parse CN, got %q", got)
	}
	// lowercase normalization
	if got := byName["ip-api"].parse("cn"); got != "CN" {
		t.Errorf("lowercase should normalize to CN, got %q", got)
	}
}
