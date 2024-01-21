# Filet Cloud Web
Web service for a minimalistic personal cloud storage, letting you control your data privacy. This has a simple and elegant design that provides a lean web interface to local storage via a local ssh connection. 

![](filet-cloud-demo.gif)

Browse files, download, upload, stream videos and music, view images, create and edit documents.

See (https://github.com/fuglaro/filet-cloud) for an example deployment on a Raspberry Pi including data snapshots. It is recommended to set up automatic backups in any deployment. 

## Supported formats
* Images
* Videos
* Audio
* PDF documents (via pdfjs)
* Markdown (with editing via easyMDE)
* Text (with editing)

Viewers and editors for new formats can be added via an internal plugin system. Plugins can be registered by file extension or content-type:
* File extension registered plugins: static/open/ext.EXTENSION.html
* Content-type registered plugins: static/open/CONTENTTYPE/SUBTYPE.html
Please get in touch if you would like any further formats supported. 

## Features
* Authentication via local user account credentials.
* Browse folder structure.
* View and edit files in supported formats.
* Stream video and audio.
* Create new folders.
* Upload files.
* Rename files and folders.
* Download files.
* Download multiple files or folders in a zip.
* Move multiple files and folders.
* Delete files and folders.
* Maintains file-system ownership integrity consistent with local access.

Automatic cloud data synchronization of smart phones and similar devices can be configured alongside this service via an ssh connection to the same server running Filet Cloud Web with apps like Folder Sync Pro. When set up securely, this can be used in place of default cloud backups allowing you full control of your personal data.

## Design and Security
Note: This was put together by someone who was usually pretty tired while coding, things will have been missed. The codebase is strikingly small and the dependencies few, so the aim is that a security audit, for whosoever whishes to do it, should be as easy as possible. Nothing is secure until it is audited and reviewed by peers.

The authentication mechanism this uses is passing ssh user and password credentials through HTTP Basic Auth to the filet-cloud-web server which uses them in accessing the SFTP server. It is therefore critical that the filet-cloud-web server is only exposed via HTTPS. It tries to not store passwords and instead relies on browser support for storing passwords to make it friendly to use.

There are some critical things to consider when making your own deployment:
* Since this uses Basic Auth to proxy ssh credentials, it is critically essential to use HTTPS if exposed to an untrusted network. HTTP is not blocked to allow for a deployment to sit behind a reverse proxy which manages TLS.
* The webserver connects to the SFTP server without verifying the ssh host key so the connection between the filet-cloud-web server and the SFTP server cannot run across an untrusted network. This project intends for the SFTP server to be on the webserver localhost itself. Connecting to localhost is hardcoded to ensure this is the case. If you change this, ensure the HostKeyCallback is changed to use something secure.
* This just acts as a proxy to a POSIX filesystem through ssh. Check your default umask. The default path is /mnt/usb/filetclouddata/-username-/. It is recommended that this have permissions of "rwx------". Users should not store data outside this folder unless their umask is suitably restrictive.

If any of this isn't clear, please do not use this if you have any data security or credential security concerns.

The code is organised in the following areas:
* [main.go](main.go) - the primary server.
* [static/storage.js](static/storagge.js) - the API given to format plugins for interacting with the storage.
* [static/browse.html](static/browse.html) - the main frontend browser page.
* [static/open/\*](static/open) - plugins for viewer and editor of different file format. Filet Cloud Web first looks for file extension matches with `static/open/ext.<file-extension>.html`, then looks for mime type matches with `static/open/<mime-type>/<sub-type>.html`, then falls back to `fallback.html`.

No frontend framework is used because adopting one on top of the simple interface design would have introduced unnecessary complexity. 

This design for this solution favors simplicity and minimalism, both inside and out, without losing powerful features. *Filet Cloud Web* pushes a personal cloud solution to its leanest essence. It is a joy to use because it does what it needs to, reliably and quickly, and then gets out of the way. The primary design philosophy for this project is: **"complexity must justify itself, ruthlessly"**.

# Installation
* Ensure your machine allows ssh from localhost.
* Build:
```bash
go build
```
* Install dependencies:
```bash
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@latest/build/pdf.min.js -O static/deps/pdf.min.js
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@latest/build/pdf.worker.min.js -O static/deps/pdf.worker.min.js
wget https://cdn.jsdelivr.net/npm/easymde/dist/easymde.min.css -O static/deps/easymde.min.css
wget https://cdn.jsdelivr.net/npm/easymde/dist/easymde.min.js -O static/deps/easymde.min.js
```
* Start server:
```bash
./filet-cloud-web
```
* Open in browser (do not connect remotely without TLS): `http://127.0.0.1:8080/`

# Thanks to, grateful forks, and contributions
We stand on the shoulders of giants. They own this, far more than I do.

* https://github.com/pkg/sftp
* https://golang.org/
* https://github.com/golang/crypto
* https://developer.mozilla.org/en-US/
* https://github.com/
* https://www.theregister.com
* https://www.nature.com/articles/s41586-021-03380-y
* https://github.com/Ionaru/easy-markdown-editor
* https://mozilla.github.io/pdf.js/
* https://www.jsdelivr.com/
* https://github.com/AlDanial/cloc
* a world of countless open source contributors.
