# sdk-gen-config


## How to Release
1. Open a PR and get your changes in main
2. Once the changes are in `main` create a tag from the main branch
   1. pull main locally: `git pull origin main`
   2. Use the command to tag : `git tag v15.48.0-pre2`
   3. Push the tag: `git push origin v15.48.0-pre2`
3. Create a release on GitHub using the new tag.
   1. go to releases
   2. click on `Draft a new release`
   3. use the tag you pushed, add release title , release notes , check set at latest release
   4. click `Publish release`