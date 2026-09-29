# Rule data

`malicious.json` is a snapshot assembled on 2026-09-30 from the OpenSSF
malicious-packages records mirrored by OSV. Every row has a direct OSV advisory
link and the publication date recorded in its note. It contains 1,510 distinct
package names across npm, PyPI, RubyGems, and crates.io. It is a local snapshot,
not a live vulnerability feed.

The checker operates on package names. It cannot distinguish a malicious version
from an unaffected version of the same package. For that reason, the snapshot
excludes `arrayref`, `internment`, and `append-only-vec` on crates.io. Their
2026-08-20 advisories concern a malicious release of an otherwise legitimate
package, and the affected release was removed. Blocking the entire name would
also block unaffected releases. See
[MAL-2026-14336](https://osv.dev/vulnerability/MAL-2026-14336) for the
`arrayref` example.

The popular-name lists are used only for edit-distance warnings. They are not
evidence that every nearby name is malicious. A typo match is a reason to stop
and review the requested package.
