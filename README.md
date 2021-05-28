# filet-cloud-web
Web portal for Filet-Cloud

This is a simple webpage exposing a cloud storage solution sitting on top of an SFTP server. It is intended to run on the same host as the SFTP server, using the SFTP server for authentication and maintaing filesystem ownership integrity consistent with local access. It indends to be elegant, simple, featureful, and a joy to use.

Please see it's parent project (https://github.com/fuglaro/filet-cloud) from which it was born.

## Supported formats
* images (browser native)
* videos (browser native)
* audio (browser native)
* pdf documents (browser native)
### Editable formats
* markdown (.md)
* text (mime text/plain)

## Features
* Authentication via local user account ssh credentials.
* Browse folder structure.
* View and edit supported files.
* Create new folders.
* Upload files.
* Rename files and folders.
* Open file in a new tab.
* Download a file.
* Download multiple files or folders in a zip.
* Move multiple files and folders.
* Delete files and folders.

## Future Work
* Ffmpeg based transcoder
* Media playlist viewer (play cart)
* View folder's thumbnails
* Change SFTP host to localhost (for security).
* Reject HTTP connections (for security).
* Write up about security.
	* Disclaimer: written when tired, nothing is secure until it is audited.
	* Uses Basic Auth to proxy ssh credentials so it is essential to use HTTPS if exposed to an untrusted network.
	* The webserver connects to the SFTP server without verifying the ssh host key, so, if running across an untrusted nwtwork, the SFTP server must be on the webserver localhost.
	* Just a proxy to POSIX through ssh. Check your default umask. The default path is /mnt/usb/filetclouddata/<username>/. It is recommended that this have permissions of "rwx------".
* File sharing: this is something not yet needed, but I'm very interested in supporting. Due to the engineering philosophy of "Complexity must justify itself", I can't add it until someone wants it. Please contact me if you would like these features. They could be something along the lines of the following:
	* Public link sharing: Creates a hard link completely open to read and/or write with a randomised name with high entropy and places it within an admin-owned folder with permissions rwx-wx-wx.
	* User sharing: Similar to public link sharing but with access fully locked except for specifically granted users given read and/or write access via ACLs (support will need to be upstreamed https://github.com/pkg/sftp).
* Office document support (spreadsheet, diagrams, slideshow, docs): This is intended but will wait and hope for the results of the recent work into getting LibreOffice supported in browsers via WebAssembly. This could result in an ideal solution compatible with this project.

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
* https://github.com/sparksuite/simplemde-markdown-editor
