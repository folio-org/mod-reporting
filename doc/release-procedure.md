# How to release a new minor version of `mod-reporting`

* `git status` to check nothing is half-resolved.
* `git pull` in case something has changed elsewhere
* `git checkout -b bX.Y`
* Edit `CHANGELOG.md` to change the "(IN PROGRESS)" date on top entry to "YYYY-MM-DD"
* `git commit -m 'Release vX.Y.0'`
* `git push --set-upstream origin bX.Y`
* Go to https://github.com/folio-org/mod-reporting/ and merge `bX.Y` into `main`.
* `git checkout main`
* `git pull`
* `git diff bX.Y` just to be sure
* `git tag vX.Y.0`
* `git push origin tag vX.Y.0`
* Go to https://github.com/folio-org/mod-reporting/ and click **Releases**, switch to the **Tags** tab, click the three-dots menu next to `vX.Y.0`, set **Release title** to be the same as the tag, paste the `CHANGELOG.md` stanza into **Release notes** and click **Publish release**
* Post the new release https://github.com/folio-org/mod-reporting/releases/tag/vX.Y.0 in the OLF Slack channel `#folio-releases`
* Go to the Jira page for vX.Y.Z (annoyingly it doesn't have a legible URL) and mark it as released.
