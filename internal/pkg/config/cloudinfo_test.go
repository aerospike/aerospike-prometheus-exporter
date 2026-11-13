package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metadataPath collapses the extra slash produced by joining the metadata root,
// which already ends in "/", with a path that also starts with "/".
func metadataPath(r *http.Request) string {
	return "/" + strings.TrimLeft(r.URL.Path, "/")
}

func useMetadataServer(t *testing.T, handler http.Handler) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cloudMetadataURL = server.URL + "/"
}

func TestCollectCloudDetailsIgnoresUnknownProvider(t *testing.T) {
	saveConfigState(t)

	Cfg.Agent.CloudProvider = " oracle "
	assert.Empty(t, CollectCloudDetails())
}

func TestCallUrl(t *testing.T) {
	saveConfigState(t)

	t.Run("returns the body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "text/html; charset=utf-8", r.Header.Get("Content-Type"))
			assert.Equal(t, "true", r.Header.Get("Metadata"))
			_, _ = w.Write([]byte("us-east-1"))
		}))
		t.Cleanup(server.Close)

		body, ok := callUrl(http.MethodGet, server.URL, map[string]string{"Metadata": "true"})
		assert.True(t, ok)
		assert.Equal(t, "us-east-1", body)
	})

	t.Run("rejects an xml error page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<?xml version="1.0"?><Error/>`))
		}))
		t.Cleanup(server.Close)

		body, ok := callUrl(http.MethodGet, server.URL, nil)
		assert.False(t, ok)
		assert.Empty(t, body)
	})

	t.Run("invalid method", func(t *testing.T) {
		body, ok := callUrl("BAD METHOD", "http://example.com", nil)
		assert.False(t, ok)
		assert.Empty(t, body)
	})

	t.Run("unreachable server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := server.URL
		server.Close()

		body, ok := callUrl(http.MethodGet, url, nil)
		assert.False(t, ok)
		assert.Empty(t, body)
	})
}

func TestAwsCloudDetails(t *testing.T) {
	saveConfigState(t)

	t.Run("region and zone", func(t *testing.T) {
		useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch metadataPath(r) {
			case "/latest/api/token":
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "21600", r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds"))
				_, _ = w.Write([]byte("tok"))
			case "/latest/meta-data/placement/region":
				assert.Equal(t, "tok", r.Header.Get("X-aws-ec2-metadata-token"))
				_, _ = w.Write([]byte("us-east-1"))
			case "/latest/meta-data/placement/availability-zone":
				assert.Equal(t, "tok", r.Header.Get("X-aws-ec2-metadata-token"))
				_, _ = w.Write([]byte("us-east-1a"))
			default:
				http.NotFound(w, r)
			}
		}))

		Cfg.Agent.CloudProvider = "aws"
		assert.Equal(t, map[string]string{
			"region":            "us-east-1",
			"availability_zone": "us-east-1a",
		}, CollectCloudDetails())
	})

	t.Run("stops when the token request fails", func(t *testing.T) {
		useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<?xml version="1.0"?><Error/>`))
		}))

		Cfg.Agent.CloudProvider = "aws"
		assert.Empty(t, CollectCloudDetails())
	})

	t.Run("keeps region when the zone request fails", func(t *testing.T) {
		useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch metadataPath(r) {
			case "/latest/api/token":
				_, _ = w.Write([]byte("tok"))
			case "/latest/meta-data/placement/region":
				_, _ = w.Write([]byte("us-east-1"))
			default:
				_, _ = w.Write([]byte(`<?xml version="1.0"?><Error/>`))
			}
		}))

		Cfg.Agent.CloudProvider = "aws"
		assert.Equal(t, map[string]string{"region": "us-east-1"}, CollectCloudDetails())
	})

	t.Run("does not replace details already collected", func(t *testing.T) {
		calls := 0
		useMetadataServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calls++
		}))
		cloudInfo = map[string]string{"region": "already"}

		getAwsCloudDetails()

		assert.Equal(t, map[string]string{"region": "already"}, cloudInfo)
		assert.Zero(t, calls)
	})
}

func TestGoogleCloudDetails(t *testing.T) {
	saveConfigState(t)

	useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/computeMetadata/v1/instance/zone", metadataPath(r))
		assert.Equal(t, "Google", r.Header.Get("Metadata-Flavor"))
		_, _ = w.Write([]byte("projects/1/zones/us-central1-a"))
	}))

	Cfg.Agent.CloudProvider = "gcp"
	assert.Equal(t, map[string]string{
		"zone": "projects/1/zones/us-central1-a",
	}, CollectCloudDetails())
}

func TestAzureCloudDetails(t *testing.T) {
	saveConfigState(t)

	useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.Header.Get("Metadata"))
		switch metadataPath(r) {
		case "/metadata/instance/compute/location":
			_, _ = w.Write([]byte("eastus"))
		case "/metadata/instance/compute/zone":
			_, _ = w.Write([]byte("1"))
		default:
			http.NotFound(w, r)
		}
	}))

	Cfg.Agent.CloudProvider = "azure"
	assert.Equal(t, map[string]string{"location": "eastus", "zone": "1"}, CollectCloudDetails())
}

func TestFetchCloudInfoMergesLabels(t *testing.T) {
	saveConfigState(t)

	useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch metadataPath(r) {
		case "/latest/api/token":
			_, _ = w.Write([]byte("tok"))
		case "/latest/meta-data/placement/region":
			_, _ = w.Write([]byte("us-east-1"))
		case "/latest/meta-data/placement/availability-zone":
			_, _ = w.Write([]byte(""))
		default:
			http.NotFound(w, r)
		}
	}))

	cfg, md := decodeConfig(t, `
[Agent]
cloud_provider = "aws"
labels = { cluster = "c1" }
`)
	Cfg = cfg
	Cfg.FetchCloudInfo(md)
	cfg = Cfg

	assert.Equal(t, "c1", cfg.Agent.MetricLabels["cluster"])
	assert.Equal(t, "us-east-1", cfg.Agent.MetricLabels["region"])
	assert.Equal(t, "null", cfg.Agent.MetricLabels["availability_zone"])
}

func TestFetchCloudInfoCreatesTheLabelMap(t *testing.T) {
	saveConfigState(t)

	useMetadataServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if metadataPath(r) == "/computeMetadata/v1/instance/zone" {
			_, _ = w.Write([]byte("projects/1/zones/us-central1-a"))
			return
		}
		http.NotFound(w, r)
	}))

	cfg, md := decodeConfig(t, `
[Agent]
cloud_provider = "gcp"
`)
	require.Nil(t, cfg.Agent.MetricLabels)
	Cfg = cfg

	Cfg.FetchCloudInfo(md)
	cfg = Cfg

	assert.Equal(t, "projects/1/zones/us-central1-a", cfg.Agent.MetricLabels["zone"])
}

func TestFetchCloudInfoSkipsWhenProviderIsUnset(t *testing.T) {
	saveConfigState(t)

	t.Run("key absent", func(t *testing.T) {
		cfg, md := decodeConfig(t, "[Agent]\n")
		cfg.Agent.MetricLabels = map[string]string{"cluster": "c1"}
		cfg.FetchCloudInfo(md)
		assert.Equal(t, map[string]string{"cluster": "c1"}, cfg.Agent.MetricLabels)
	})

	t.Run("blank value", func(t *testing.T) {
		cfg, md := decodeConfig(t, "[Agent]\ncloud_provider = \"   \"\n")
		cfg.FetchCloudInfo(md)
		assert.Nil(t, cfg.Agent.MetricLabels)
	})
}
