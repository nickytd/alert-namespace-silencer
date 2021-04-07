package silencer

import (
	"fmt"
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

	v := fmt.Sprintf("%s/%s", ".*", value)
	if m, err = labels.NewMatcher(labels.MatchEqual, name, v); err != nil {
		klog.ErrorS(err, "cannot create matcher")
		return false
	}

	param := silence.NewGetSilencesParams().
		WithFilter([]string{m.String()})

	return deleteSilences(param)
}

func deleteSilences(param *silence.GetSilencesParams) bool {
	var err error
	var res *silence.GetSilencesOK
	var deleted bool

	if res, err = manager.Silence.GetSilences(param); err != nil {
		klog.ErrorS(err, "cannot get silences")
		return false
	}

	for _, s := range res.GetPayload() {
		uid := strfmt.UUID(*s.ID)

		if models.SilenceStatusStateExpired == *s.Status.State {
			klog.V(4).InfoS("skip expired silence", "id", uid)
			continue
		}

		if *s.CreatedBy != createdby {
			klog.V(4).InfoS(
				"skip unmanaged silence",
				"id", uid,
				"createdBy", *s.CreatedBy,
			)
			continue
		}

		deleteParam := silence.NewDeleteSilenceParams().WithSilenceID(uid)
		if _, err = manager.Silence.DeleteSilence(deleteParam); err != nil {
			klog.ErrorS(
				err, "error deleting silence",
				"id", uid.String(),
			)
			deleted = false
		} else {
			klog.V(2).InfoS(
				"deleting silence",
				"id", uid.String(),
			)
			deleted = true
		}
	}
	return deleted
}

func createSilencer(namespace string, value string) *models.PostableSilence {

	now := time.Now().UTC()
	starts := strfmt.DateTime(now)
	ends := strfmt.DateTime(now.AddDate(0, 1, 0))

	regex := true
	v := fmt.Sprintf("%s/%s", ".*", value)
	matcher := models.Matcher{
		IsRegex: &regex,
		Name:    &namespace,
		Value:   &v,
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

func CleanSilences() {
	if deleteSilences(silence.NewGetSilencesParams()) {
		klog.InfoS("silence clean up finished successfully")
	}
}
