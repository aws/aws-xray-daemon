# X-Ray daemon — S3 monitoring alarms

This document defines the CloudWatch alarms for the S3 / tag-alias monitoring
added to `continuous-monitoring.yml` and `install-smoke-test.yml`. It follows
how X-Ray daemon alarms are managed today: **out-of-band** (created directly in
CloudWatch, not as code in this repo), leaf alarms with empty actions that are
wired to paging through the central `[AutoCut] X-Ray SDK Distribution Channel
Monitor` composite alarm.

All metrics are published to the `MonitorDaemon` namespace in **us-east-1** and
are **not** regionalized (per-region results are rolled up into a count on a
single metric), so the alarms are single-region too — matching the existing
`PullFailureFromECR` / `StartupFailureFromECR` alarms.

## Existing convention (for reference)

The current leaf alarms (e.g. `StartupFailureFromECR`) use:

- Threshold: `> 0.1`, comparison `GreaterThanThreshold`
- Statistic `Average`, period `600`, evaluation periods `3`
- `treatMissingData: missing`
- Empty `AlarmActions`; paging is via the `[AutoCut]` composite alarm's OR-rule.

## New leaf alarms

One alarm per metric, aggregated across the `artifact` / `channel` dimensions
(the run log names the specific artifact/region; the alarm names what failed).
Aggregation uses metric math (`SUM` of all dimensioned samples for the metric),
so any artifact failing drives the alarm.

| Alarm name | Metric (namespace `MonitorDaemon`) | Aggregation | Period | Eval periods | treatMissingData | Notes |
|---|---|---|---|---|---|---|
| `S3DownloadFailure` | `DownloadFailureFromS3` | SUM across `artifact` | 600 | 3 | `missing` | 10-min job. Value = regions failing; any sustained non-zero pages. |
| `S3SignatureVerificationFailure` | `SignatureVerificationFailureFromS3` | SUM across `artifact` | 600 | 3 | `missing` | 10-min job. |
| `S3TagAliasMismatch` | `TagAliasMismatchFromECR` + `FromDockerhub` + `FromS3` | SUM across the 3 metrics/channels | 600 | 3 | `missing` | 10-min job. One alarm covers all three channels. |
| `S3PackageInstallFailure` | `PackageInstallFailureFromS3` | SUM across `artifact` | **3600** | 2 | `missing` | **Hourly** job — period must be 3600, not 600, or 5 of 6 datapoints are missing and the alarm never evaluates correctly. |

Threshold for all: `> 0.1`, `GreaterThanThreshold` (metric is a count / 0-1, so
`> 0.1` means "at least one failure"). Statistic: the metric-math expression
returns the SUM; alarm on that expression.

### Example metric-math alarm (S3DownloadFailure)

```
aws cloudwatch put-metric-alarm \
  --alarm-name S3DownloadFailure \
  --alarm-description "An X-Ray daemon artifact failed to download from S3 (see the monitor-s3 job log for the artifact/region)." \
  --namespace MonitorDaemon \
  --comparison-operator GreaterThanThreshold \
  --threshold 0.1 \
  --evaluation-periods 3 \
  --treat-missing-data missing \
  --metrics '[
    {
      "Id": "e1",
      "Expression": "SUM(SEARCH('"'"'{MonitorDaemon,failure,artifact} MetricName=\"DownloadFailureFromS3\"'"'"', '"'"'Average'"'"', 600))",
      "Label": "DownloadFailuresAllArtifacts",
      "ReturnData": true
    }
  ]'
```

(The same shape applies to the other three, changing `MetricName`, period, and
alarm name. `S3TagAliasMismatch` searches all three `TagAliasMismatchFrom*`
metric names.)

## Wiring to paging

Leaf alarms above have no direct actions (matching the existing daemon alarms).
Add each to the OR-rule of the existing composite alarm **`[AutoCut] X-Ray SDK
Distribution Channel Monitor`**, which holds the ticketing action
(`arn:aws:cloudwatch::cwa-internal:ticket:3:AWS:X-Ray:Operations:...`):

```
... existing rule ... OR
ALARM("S3DownloadFailure") OR
ALARM("S3SignatureVerificationFailure") OR
ALARM("S3TagAliasMismatch") OR
ALARM("S3PackageInstallFailure")
```

Optionally also create `InsufficientData` alarms (as the SDKs do, e.g.
`XRayGoSDKInsufficientData`) so that a monitor that stops running entirely is
itself caught — the leaf alarms use `treatMissingData: missing`, which will not
page on its own.

## Coverage audit — every metric has an alarm

| Metric | Covered by |
|---|---|
| `DownloadFailureFromS3` | `S3DownloadFailure` |
| `SignatureVerificationFailureFromS3` | `S3SignatureVerificationFailure` |
| `TagAliasMismatchFromECR` | `S3TagAliasMismatch` |
| `TagAliasMismatchFromDockerhub` | `S3TagAliasMismatch` |
| `TagAliasMismatchFromS3` | `S3TagAliasMismatch` |
| `PackageInstallFailureFromS3` | `S3PackageInstallFailure` |

## Testing the alarms (per the plan's Phase 5 test strategy)

- **Alarm-fires:** temporarily drive one metric to 1 (e.g. point one artifact at
  a bad key on a branch run, or `put-metric-data ... --value 1`) and confirm the
  leaf alarm goes to ALARM and the `[AutoCut]` composite cuts a ticket in a
  test/staging destination; then restore and confirm it returns to OK.
- **Missing-data:** stop emitting a metric and confirm behavior matches the
  chosen `treatMissingData` (and that any `InsufficientData` alarm catches a
  monitor that stopped running).
- **Period correctness:** confirm `S3PackageInstallFailure` uses 3600s — with
  600s, the hourly metric leaves 5 of 6 datapoints missing each period.
