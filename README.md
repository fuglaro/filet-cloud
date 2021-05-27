# filet-cloud-web
Web portal for Filet-Cloud

This is a simple webpage exposing a cloud storage solution, which sits on top of an SFTP server.

## Future Work
* More content viewers (with editors):
	* md and other text files
	* spreadsheet
	* diagrams
	* slideshow
	* docs
	* pdf
* content editors
* Ffmpeg based transcoder
* Media playlist viewer (play cart)
* open content in new tab (without index) http://www.zuga.net/articles/unicode/character/1F4D6/
* delete items (with confirmation) http://www.zuga.net/articles/unicode/character/1F5D1/
* Share files via secure link (via making public to the pi user and thus the webserver in a PUBLIC folder). Ensure directory above is not readable but is executable. Check that is actually secure.
* Change SFTP host to localhost (for security).
* Reject HTTP connections (for security).
* Write up about security.
	* Disclaimer: written when tired, nothing is secure until it is audited.
	* Uses Basic Auth to proxy ssh credentials so it is essential to use HTTPS if exposed to an untrusted network.
	* The webserver connects to the SFTP server without verifying the ssh host key, so, if running across an untrusted nwtwork, the SFTP server must be on the webserver localhost.

## Design and Engineering Philosophies

This project explores how far a software product can be pushed in terms of simplicity and minimalism, both inside and out, without losing powerful features. Web programs and cloud tools tends to be bloated and buggy, as all software tends to be. *filetcloud* pushes a personal cloud solution to its leanest essence. It is a joy to use because it does what it needs to, reliably and quickly, and tries to do nothing else. The opinions that drove the project are:

* **Complexity must justify itself**.
* Lightweight is better than heavyweight.
* Select your dependencies wisely: they are complexity, but not using them, or using the wrong ones, can lead to worse complexity.
* Powerful features are good, but simplicity and clarity are essential.
* Adding layers of simplicity, to avoid understanding something useful, only adds complexity, and is a trap for learning trivia instead of knowledge.
* Steep learning curves are dangerous, but don't just push a vertical wall deeper; learning is good, so make the incline gradual for as long as possible.
* Allow other tools to thrive - e.g: terminals don't need tabs or scrollback, that's what tmux is for.
* Fix where fixes belong - don't work around bugs in other applications, contribute to them, or make something better.
* Improvement via reduction is sometimes what a project desperately needs, because we do so tend to just add. (https://www.theregister.com/2021/04/09/people_complicate_things/, https://www.nature.com/articles/s41586-021-03380-y)

# Thanks to, grateful forks, and contributions

We stand on the shoulders of giants. They own this, far more than I do.

* https://golang.org/
* https://developer.mozilla.org/en-US/
* https://github.com/
* https://www.theregister.com
* https://www.nature.com/articles/s41586-021-03380-y
* https://mozilla.github.io/pdf.js/ // TODO remove in favor of browser
