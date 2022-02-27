# filet-cloud-web
Web portal for Filet-Cloud

![](filet-cloud-demo.gif)

This is a simple webpage exposing a cloud storage solution sitting on top of an SFTP server. It intends to be elegant, simple, featureful, and a joy to use. Breathtaking simplicity is one of the core driving principles.

Please see it's parent project (https://github.com/fuglaro/filet-cloud) from which it was born. That includes the best information for deployment.

## Supported formats
* Images (browser native)
* Videos (browser native)
* Audio (browser native)
* PDF documents (pdfjs)
* Markdown (\*.md simpleMDE - editable)
* Text (mime text/plain - editable)

Please get in touch if you would like any further formats supported. Frontend viewers and editors can easily be added via a plugin system registered by file extension or content-type:
* File extension registered plugins: template/open/ext.EXTENSION.html
* Content-type registered plugins: template/open/CONTENTTYPE/SUBTYPE.html

## Features
* Authentication via local user account ssh credentials.
* Browse folder structure.
* View and edit supported files.
* Stream video and audio.
* Create new folders.
* Upload files.
* Rename files and folders.
* Open file in a new tab.
* Download a file.
* Download multiple files or folders in a zip.
* Move multiple files and folders.
* Delete files and folders.
* Maintains filesystem ownership integrity consistent with local access.

## Rationale for Omissions
* Video transcoding for playback of old codecs in modern browsers - I threw this together with ffmpeg and a simple streaming approach and it was slow and didn't allow seeking. Fixing those issues would mean buffering large amounts of memory or transcoding the entire video to disk ahead of serving. A simpler and cleaner approach is just to expect users to keep video files up to date with modern codecs. This is better in the long run. Leaving the poorly supported or old video formats around will just make them age harder, better to rip from VHS than to keep a VHS player around forever.
* WebDAV support - This, I must admit, is enticing. Especially when a meaty portion of this codebase is essentially exposing a filesystem through HTTP. It has not been pursued because it would add significant levels of complexity to support the specification fully, and goes against the original aims of the project - to expose a webUI to an SFTP connection. If we have chosen SFTP, we already have an API interface to the cloud storage, and supporting multiple interfaces is just complexity. Anything else can be a fork, which I welcome.
* Office document support - This is planned. See Future Work.

## Security
Note: This was put together by someone who was usually pretty tired while coding, things will have been missed. The codebase is strikingly small and the dependencies few, so the aim is that a security audit, for whosoever whishes to do it, should be as easy as possible. Nothing is secure until it is audited and reviewed by peers.

The authentication mechanism this uses is passing ssh user and password credentials through HTTP Basic Auth to the filet-cloud-web server which uses them in accessing the SFTP server. It is therefore critical that the filet-cloud-web server is only exposed via HTTPS. It tries to not store passwords and instead relies on browser support for storing passwords to make it friendly to use.

There are some critical things to consider when making your own deployment:
* Since this uses Basic Auth to proxy ssh credentials, it is critically essential to use HTTPS if exposed to an untrusted network. HTTP is not blocked to allow for a deployment to sit behind a reverse proxy which manages TLS.
* The webserver connects to the SFTP server without verifying the ssh host key so the connection between the filet-cloud-web server and the SFTP server cannot run across an untrusted network. This project intends for the SFTP server to be on the webserver localhost itself. Connecting to localhost is hardcoded to ensure this is the case. If you change this, ensure the HostKeyCallback is changed to use something secure.
* This just acts as a proxy to a POSIX filesystem through ssh. Check your default umask. The default path is /mnt/usb/filetclouddata/-username-/. It is recommended that this have permissions of "rwx------". Users should not store data outside this folder unless their umask is suitably restrictive.

If any of this isn't clear, please do not use this if you have any data security or credential security concerns.

## Future Work
* FFmpeg is used to create the thumbnails for images as well as videos but doesn't respect EXIF metadata, sometimes resulting in rotated thumbnails. I'd like to look into upstreaming this to ffmpeg, if they would have it. It also doesn't catch all image types, this could be expanded upon by utilising other shell tools.
* File sharing: this is something not yet needed, but I'm very interested in supporting. Due to the engineering philosophy of "Complexity must justify itself", I can't add it until someone wants it. I haven't needed it yet. Please contact me if you would like these features. They could be something along the lines of the following:
	* Public link sharing: Creates a hard link completely open to read and/or write with a randomised name with high entropy and places it within an admin-owned folder with permissions rwx-wx-wx.
	* User sharing: Similar to public link sharing but with access fully locked except for specifically granted users given read and/or write access via ACLs (support will need to be upstreamed https://github.com/pkg/sftp).
* Office document support (spreadsheet, diagrams, slideshow, docs): This is intended but will wait and hope for the results of the recent work into getting LibreOffice supported in browsers via WebAssembly. This could result in an ideal solution compatible with this project. I look forward to supporting this in the future.

## Code Layout

Cloc'ing in at under 500 lines of code, plus a lean count of dependencies, there is not much to this. The code is separated into the following areas:
* [main.go](https://github.com/fuglaro/filet-cloud-web/blob/main/main.go) - the primary server dishing out frontend html and Javascript and fielding WebAPI requests to interact with the SFTP server.
* [template/main.html](https://github.com/fuglaro/filet-cloud-web/blob/main/template/main.html) - the HTML for the main frontend browser page.
* [static/main.js](https://github.com/fuglaro/filet-cloud-web/blob/main/static/main.js) - the Javascript for the main frontend browser page.
* [template/open/\*](https://github.com/fuglaro/filet-cloud-web/tree/main/template/open) - the HTML for plugin viewers and editors for different file types. First looks for file extension matches with `template/open/ext.<file-extension>.html`, then looks for mime type matches with `template/open/<mime-type>/<sub-type>.html`, then falls back to `fallback.html`. Templates are all resolved with `{{.P}}` as the path of the file to open, and `{{.M}}` as the detected MIME type.
* [static/open/\*](https://github.com/fuglaro/filet-cloud-web/tree/main/static/open) - the Javascript and other static files for the plugin viewers and editors.

Why no fancy frontend framework? - The design of the code and tool is too simple to justify any need. Frameworks should facilitate simplification, and this is already simple.

## Design and Engineering Philosophies
This project explores how far a software product can be pushed in terms of simplicity and minimalism, both inside and out, without losing powerful features. Web programs and cloud tools tend to be bloated and buggy, as all software tends to be. *filetcloud* pushes a personal cloud solution to its leanest essence. It is a joy to use because it does what it needs to, reliably and quickly, and tries to do nothing else. The opinions that drove the project are:

* **Complexity must justify itself**.
* Lightweight is better than heavyweight.
* Select your dependencies wisely: they are complexity, but not using them, or using the wrong ones, can lead to worse complexity.
* Powerful features are good, but simplicity and clarity are essential.
* Adding layers of simplicity, to avoid understanding something useful, only adds complexity, and is a trap for learning trivia instead of knowledge.
* Steep learning curves are dangerous, but don't just push a vertical wall deeper; learning is good, so make the incline gradual for as long as possible.
* Allow other tools to thrive - e.g: terminals don't need tabs or scrollback, that's what tmux is for.
* Fix where fixes belong - don't work around bugs in other applications, contribute to them, or make something better.
* Improvement via reduction is sometimes what a project desperately needs, because we do so tend to just add. (https://www.theregister.com/2021/04/09/people_complicate_things/, https://www.nature.com/articles/s41586-021-03380-y)

# Development Testing

Note!: Do not use this in a production or in any untrusted environment as this setup bypasses security protections.

* Ensure your development machine allows ssh from localhost.
* Build:
```bash
go build
```
* Download dependencies:
```bash
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@latest/build/pdf.min.js -O static/pdf.min.js
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@latest/build/pdf.worker.min.js -O static/pdf.worker.min.js
wget https://cdn.jsdelivr.net/simplemde/latest/simplemde.min.css -O static/simplemde.min.css
wget https://cdn.jsdelivr.net/simplemde/latest/simplemde.min.js -O static/simplemde.min.js
```
* Start server:
```bash
FILETCLOUDDIR=/home ./filet-cloud-web
```
* Open in browser (insecurely): `http://127.0.0.1:8080/?P=/`

# Thanks to, grateful forks, and contributions
We stand on the shoulders of giants. They own this, far more than I do.

* https://github.com/pkg/sftp
* https://golang.org/
* https://github.com/golang/crypto
* https://developer.mozilla.org/en-US/
* https://github.com/
* https://www.theregister.com
* https://www.nature.com/articles/s41586-021-03380-y
* https://github.com/sparksuite/simplemde-markdown-editor
* https://mozilla.github.io/pdf.js/
* https://www.jsdelivr.com/
* https://github.com/AlDanial/cloc
* a world of countless open source contributors.
