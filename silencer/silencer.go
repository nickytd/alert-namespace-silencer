package silencer

import (
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/pkg/labels"
	"k8s.io/klog/v2"
	"net/url"
	"time"
)

var (
	comment   = "automated silencer"
	createdby = "alert-namespace-silencer"
)

var manager *client.Alertmanager

func InitAlertManager(url url.URL) {
	cfg := &client.TransportConfig{}
	config := cfg.WithHost(url.Host).
		WithSchemes([]string{url.Scheme}).
		WithBasePath("/api/v2")
	manager = client.NewHTTPClientWithConfig(nil, config)
}

func AddSilencer(name string, value string) bool {

	s := createSilencer(name, value)

	param := silence.NewPostSilencesParams().WithSilence(s)
	ok, err := manager.Silence.PostSilences(param)
	if err != nil {
		klog.ErrorS(err, "error creating silence")
		return false
	}
	res := ok.GetPayload().SilenceID
	klog.InfoS("creating silence", "id", res)
	return true
}

// we get all silencers and remove those with matching name and value
func RemoveSilencer(name string, value string) bool {
	var m *labels.Matcher
	var err error
	var res *silence.GetSilencesOK
	var dres *silence.DeleteSilenceOK

	if m, err = labels.NewMatcher(labels.MatchEqual, name, value); err != nil {
		klog.ErrorS(err, "cannot create matcher")
		return false
	}

	param := silence.NewGetSilencesParams().
		WithFilter([]string{m.String()})

	if res, err = manager.Silence.GetSilences(param); err != nil {
		klog.ErrorS(err, "cannot get silences")
		return false
	}

	for _, s := range res.GetPayload() {
		uid := strfmt.UUID(*s.ID)
		klog.V(4).InfoS(
			"silence",
			"id", uid,
			"status", *s.Status.State,
		)

		if models.SilenceStatusStateExpired == *s.Status.State {
			continue
		}

		deleteParam := silence.NewDeleteSilenceParams().WithSilenceID(uid)
		if dres, err = manager.Silence.DeleteSilence(deleteParam); err != nil {
			klog.ErrorS(
				err, "error deleting silence",
				"id", uid.String(),
			)
		} else {
			klog.V(2).InfoS(
				"deleting silence", "result",
				dres.Error(),
			)
		}
	}

	return true
}

func createSilencer(namespace string, value string) *models.PostableSilence {

	now := time.Now().UTC()
	starts := strfmt.DateTime(now)
	ends := strfmt.DateTime(now.AddDate(1, 0, 0))

	regex := false

	matcher := models.Matcher{
		IsRegex: &regex,
		Name:    &namespace,
		Value:   &value,
	}

	matchers := models.Matchers{
		&matcher,
	}

	s := models.Silence{
		Comment:   &comment,
		CreatedBy: &createdby,
		EndsAt:    &ends,
		Matchers:  matchers,
		StartsAt:  &starts,
	}

	postableSilence := models.PostableSilence{
		Silence: s,
	}

	return &postableSilence
}
