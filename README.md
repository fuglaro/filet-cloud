==== TODO ====
==== TODO ====
==== TODO ====

# SECURITY

Since this service proxies SSH credentials, and both serves and modifies personal data,
strict security policies have been implemented. Please use a modern and up-to-date browser
and device to make full use of these protections.

## Transport Security
* The backend implements a strict HTTPS ONLY policy. TODO (check for static and resources, fix localhost omissions)
* HTTP requests to the main page are redirected to HTTPS. TODO
* HTTP Strict Transport Security (HSTS) is enforced. TODO
* All WebSocket connections use WebSocket Secure (WSS). TODO
* The Content Security Policy is configured to ensure that content is only loaded via HTTPS. TODO add https: to all parts of the content security policy.

## Authentication
* Authentication is made by proxying the SSH authentication mechanism through the backend in establishing a local SSH connection managed by the backend.
* This primarily relies on SSH username and password authentication.
* 2FA can be additionally configured with a Pluggable Authentication Module (PAM). TODO https://www.digitalocean.com/community/tutorials/how-to-set-up-multi-factor-authentication-for-ssh-on-ubuntu-20-04

## Login Session Managment
* On completion of the login form, an authenticated secure connection is established:
  * A WebSocket connection is established with the backend (with WSS).
  * The login credentials are passed directly to the WebSocket connection.
  * The backend passes the credentials directly into establishing an SFTP/SSH connection locally.
  * The SFTP/SSH connection is attached to the WebSocket connection to handle future requests.
  * The credentials are not stored in any persistent way.
  * Failure to establish an authenticated SFTP/SSH connection will close the WebSocket connection, triggering a new login sequence.
  * After sending the credentials to the WebSocket connection, the login form will pass the potentially authenticated WebSocket connection to be stored inside an instance of the Storage class in a private variable, so as to restrict direct access from JavaScript except via the Storage class' API.
* Logout occurs when either the browser or the backend closes the WebSocket connection, such as:
  * Automatically when closing or refreshing the browser tab.
  * When restarting the backend service.
  * From a disruption to the network connection.
* Logout events will trigger all cached site data to be cleared. TODO https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Clear-Site-Data
  * Cached site data may not be cleared if the browser exits uncleanly.
* The user may choose to store the credentials in the browser's password management system, if supported and enabled in the browser. For additial security, 2FA is recommended. TODO

## Site Isolation and Content Protection
* Same-Origin Policy is enforced. TODO (https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy)
* Content Security Policy is configured to only allow content coming from the site's own origin. TODO (https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
* Cross Origin Isolation is enforced by: TODO
  * Setting the Cross Origin Opener Policy to ensure the browsing context is exclusively isolated to same-origin documents. TODO: All: Cross-Origin-Opener-Policy: same-origin
  * Setting the Cross Origin Embedder Policy to require corp (Cross Origin Resource Policy). TODO
  * Ensuring Cross Origin Isolation is fully activated by checking that the crossOriginIsolated property in the browser is active, before opening the login form. TODO  (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cross-Origin-Embedder-Policy To check if cross origin isolation has been successful, you can test against the crossOriginIsolated property available to window and worker contexts: )
* Default Cross Origin Read Blocking browser protections are enhanced by all Content Type Options being configured with nosniff, and the Content-Type header being set based on file name extension TODO or inspection of the first block. TODO X-Content-Type-Options: nosniff
* Cross Origin Resource Policy is configured to same-origin so that all resources are protected from access by any other origin. TODO ALL: Cross-Origin-Resource-Policy: same-origin
* Content Security Policy is enforced with a configuration that ensures: TODO (https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
  * Image and media content can only be loaded from the site's own origin. TODO
  * Script, stylesheet, and font resources can only be loaded from the site's own origin. TODO
  * Contents that do not match the above types, are denied. TODO
  * All content is loaded sandboxed with restricted allowances. TODO (omit sandbox for browse.html)
  * Documents are prevented from being embedded. TODO
  * Forms are denied from using URLs as the target of form submission. TODO
  ``` browse.html
  Content-Security-Policy:
   sandbox allow-downloads allow-forms allow-same-origin allow-scripts;
   default-src 'none';
   frame-ancestors: 'none';
   form-action: 'none';
   image-src 'self';
   media-src 'self';
   script-src-elem 'self';
   style-src 'self';
   font-src 'self';
  TODO
  ```
  ``` everything else
  Content-Security-Policy:
   sandbox;
   default-src 'none';
   frame-ancestors: 'none';
   form-action: 'none';
  TODO
  ```
* The backend requires the browser to provide Secure Fetch Metadata Request Headers, and denies access to content unless the following policies are met: TODO
  * For the main page:
    * The request site is 'none', ensuring user initiated access.
    * The request mode is not from a navigate, preventing access through links.
    * The request destination is a document, preventing embedding.
  * For the /static/deps/ URL path:
    * The request site and mode is same-origin.
    * The request destination is a script, font or style.
  * For the /connect endpoint:
    * The request site and mode is same-origin.
    * The request destination is a websocket.
  * For the /authenticate endpoint:
    * The request site and mode is same-origin.
    * The request destination is empty.
  * For everything else:
    * The request site and mode is same-origin.
    * The request destination is audio, an image, or a video.
* The backend enforces a browser cache policy which ensures cached content access adheres to the above Secure Fetch Metadata Request Header policy, including when the headers vary across subsequent requests. TODO Vary: Sec-Fetch-Dest, Sec-Fetch-Mode, Sec-Fetch-Site
* The browser is instructed to not allow content to be loaded in any embeded documents, by setting X-Frame-Options: DENY. TODO X-Frame-Options: DENY
* A Referrer Policy of same-origin is enforced. TODO

## Authorised Browser Access to Content
* Along with storage requests being served via the authenticated WebSocket connection, time limited authorised access can be extended to same-origin access by the browser for the purpose of allowing the display of media and content in the browser, and for downloads. TODO
* Extension of authorised access to the browser is achieved via the following process: TODO
  * A link for storage content is requested via the API of the Storage class.
  * Before returning the link, the Storage class, if it has not already done so in the previous 2 seconds, will: TODO
    * Request a JSON Web Token (JWT) for authorization, via the authenticated Websocket connection, which is created by the backend with the following JWT payload: TODO https://datatracker.ietf.org/doc/html/rfc7519#section-4.1
      * The remote address IP (which is never stored persistently by the backend) of the authenticated WebSocket connection's client side as the Registered Audience Claim (aud). TODO
      * A time of 5 seconds later as the Registered Expiration Time Claim (exp). TODO
      * A cryptographically secure pseudorandom number associated with the authenticated WebSocket connection as the Registered Subject Claim (sub). TODO
    * Prevent exposure of the authorization JWT to Javascript contexts outside the Storage class. TODO
    * Send the authorization JWT, via request body, to the backend's /authenticate endpoint so it can instruct the browser to set the autherization JWT as an authentication cookie with the following cookie attribute protections: TODO https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-User
       * The browser only uses the authentication cookie in requests back to the site's own originating site, by setting SameSite=Strict. TODO
       * The browser expires the cookie after 5 seconds, by setting Max-Age=5. TODO
       * The authorization JWT is protected from JavaScript access, by setting HttpOnly. TODO
       * The cookie is further protected by setting the Secure cookie attribute, and by giving the cookie name the secure __Host- prefix. TODO
  * Returns the link the requested via the Storage class API.
  * Requests to any storage link will succeed only if the backend's checks of the authorization JWT cookie is successfully validated with the following policy: TODO
    * The remote address IP of the request must match the JWT's Registered Audience Claim. TODO
    * The JWT's Registered Expiration Time Claim must not have expired. TODO
    * The JWT's Registered Subject Claim (sub) must exactly match a previously generated cryptographically secure pseudorandom number associated with an authenticated WebSocket connection, which is then used to fulfill the storage request. TODO

## Additional Cross-Site Request Forgery (CSRF/XSRF) Protection
* All backend endpoints which cause any changes or side effects (besides server load), are only accessible through the WebSocket connection. TODO
* The WebSocket connection is stored in a private variable, inside the Storage class, and is only accessible via it's restricted API.
* No cross-site queries have access to that WebSocket connection.

## Third-Party Dependencies
* All third-party dependencies are servered from the backend and are version controlled and stored locally.
* All third-party dependencies loaded in the browser are Subresource Integrity checked. TODO

== TODO
* https://web.dev/articles/coop-coep
* https://web.dev/articles/cross-origin-isolation-guide
* https://web.dev/articles/why-coop-coep
* https://web.dev/articles/fetch-metadata
* https://xsleaks.dev/docs/defenses/isolation-policies/resource-isolation/
* https://www.chromium.org/Home/chromium-security/site-isolation/
* https://chromium.googlesource.com/chromium/src/+/master/services/network/cross_origin_read_blocking_explainer.md#determining-whether-a-response-is-corb_protected
* https://www.w3.org/TR/post-spectre-webdev/#documents-isolated
* https://w3c.github.io/webappsec-fetch-metadata/
* https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy
* https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/sandbox
* https://developer.mozilla.org/en-US/docs/Web/HTTP/Cross-Origin_Resource_Policy
* https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cross-Origin-Embedder-Policy

* https://docs.google.com/document/d/1zDlfvfTJ_9e8Jdc8ehuV4zMEu9ySMCiTGMS9y0GU92k/edit
* Read over the security section to check it.

* Test favicon still works after security changes.
* TODO Switch to package managed dependencies and update security docs -- maybe.
* TODO Run some standard test suites.
* TODO Setup security regression tests.

* Kill Basic Auth

* Launch thumbnail loads at the same time as file loads but always let the file win.
* Support other image types for thumbnails smartly.
* Abort early on getting thumbnail link that wont work.
* Don't try to query the thumbnail if the full image is already cached by checking the .complete property on the image without waiting.
* When the links ensure authentication, allow them to be awaited on.
* Investigate hardware accelerated thumbnail generation.
* Reload files without loging in again.
* Auto reload unmodified opened files.
* roll our own popup - nobody likes browser pop ups.
* ensure blur in load until confirmed login - especially between login attempts.
* dark mode try 2.
* best practise on inline unicode symbols
* Switch relevant divs to buttons, ensure accessibility, and check keyboard only input.
* xtermjs via same WebSocket connection that allows sshConn endpoint including resize triggers, distinguish from uploads with first bit.
* Document env vars (be clear about cloud dir needing username folder after that level).
* Deliver system information to file (accessible via xtermjs), so a HAT is not needed.

* Add support for a different local ssh port (Android Termux)

* Swap to RP3A+ for low power and compact.
* Mobile phone version with battery pack and 4G for always on.

* Monitor storage interval access (maybe backup interval is an issue - maybe only backup when things change) for allowing the storage to power down.

* note on working well with: https://stephango.com/file-over-app
* Document about embedding resources in Markdown files.
* Make as a Progressive Web App (PWA)

# Filet Cloud Web
Web service for a minimalistic personal cloud storage, letting you control your data privacy. This has a simple and elegant design that provides a lean web interface to local storage via a local ssh connection. 

![demo video](filet-cloud-demo.gif)

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
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@3.11.174/build/pdf.min.js -O static/deps/pdf.min.js
wget https://cdn.jsdelivr.net/npm/pdfjs-dist@3.11.174/build/pdf.worker.min.js -O static/deps/pdf.worker.min.js
wget https://cdn.jsdelivr.net/npm/easymde@2.18.0/dist/easymde.min.css -O static/deps/easymde.min.css
wget https://cdn.jsdelivr.net/npm/easymde@2.18.0/dist/easymde.min.js -O static/deps/easymde.min.js
```
* Start server:
```bash
./filet-cloud-web
```
* Open in browser (do not connect remotely without TLS): `http://127.0.0.1:8080/`

# Thanks to
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
