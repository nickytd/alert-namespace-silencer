# alert-namespace-silencer
A controller creating alertmanger silences based on namespace labels.
It leverages [K8S shared informers](https://www.cncf.io/blog/2019/10/15/extend-kubernetes-via-a-shared-informer/) watching for namespace updates and Alertmanager [Rest Api](https://github.com/prometheus/alertmanager/blob/master/api/v2/openapi.yaml)

If namespace does not have an enable-alert label a silence is created.
Example for a silencer related to alerts originating from default namespace
```
{
        "id": "241425f3-6537-4f55-8633-db09f7b8f839",
        "status": {
            "state": "active"
        },
        "updatedAt": "2021-03-21T10:48:25.316Z",
        "comment": "automated silencer",
        "createdBy": "alert-namespace-silencer",
        "endsAt": "2022-03-21T10:48:25.115Z",
        "matchers": [
            {
                "isRegex": false,
                "name": "namespace",
                "value": "default"
            }
        ],
        "startsAt": "2021-03-21T10:48:25.316Z"
    }
```

Both namespace label and silence matcher names are configurable
