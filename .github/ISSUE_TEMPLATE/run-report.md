---
name: Run report
about: Share an anonymized summary of a noCRUD run
title: "[run] "
labels: run-report
---

<!--
Thanks for sharing a run — this is how the project learns what people actually
point noCRUD at, and at what size.

EVERYTHING HERE IS PUBLIC. Please read the JSON below before submitting.

The `nocrud-share-results` skill generates this payload for you and shows it to
you first. If you are filling this in by hand, the rules are:

  INCLUDE  runner, counts, timings, mode, concurrency, your framework/language,
           OS, CPU count, postgres version
  NEVER    flow names, endpoint names, field names, model names, URLs,
           hostnames, database names, your git sha, usernames, fixture contents

Flow and endpoint names are the ones that catch people out — they usually ARE
your domain model ("claim", "patient", "invoice"). Counts, not names.
-->

## Payload

```json
PASTE THE GENERATED JSON HERE
```

## Anything you want to add? (optional, free text)

<!--
Entirely optional. Useful things: what kind of backend it is, whether the setup
was painful anywhere, whether the numbers surprised you. Please keep it free of
anything identifying — same rules as above.
-->

## Confirmation

- [ ] I have read the JSON above and I am happy for it to be public.
