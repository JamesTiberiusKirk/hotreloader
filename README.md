# HotReloader

HotReloading middleware for golang web applications (currectly supporting only fiber) which watches tempalte folders (recusively) and automatically reload web pages when they are modified.

API will be chainging as i add more configuration options and make it more cross platform (so it does not just work on fiber)

More info to come...

## Bugs

- If you make any new folders in any of the watched folders, you might need to reload the server as by default it will not be watched
- For some reason the injected script does not actually get injected at the bottom of the body
- Might be related to the above...but the injector seems to run on any http response, not just the ones which contain a "frame"
  - so basically i want it to not be injected unless its a full page being sent down rather than just individual "htmx" components
