==== TODO ====
==== TODO ====
==== TODO ====

== TODO
* https - accept certs via env var or auto setup with let's encrypt autocert NewListener (with domain provided by FC_DOMAIN) and note on acceptance of LetsEncrypt Terms of Service.
* Add underscores to other env vars.

* Test favicon still works after security changes.
* TODO Switch to package managed dependencies and update security docs -- maybe.
* TODO Run some standard test suites.
* TODO Setup security regression tests.
* Rename browse.html to main.html (clearer compliance with the security documentation).

* Kill Basic Auth
* When the links ensure authentication, allow them to be awaited on.

* Reload files without loging in again.
* Auto reload unmodified opened files.
* roll our own popup - nobody likes browser pop ups.
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
* Thumbnails
  * Support other image types for thumbnails smartly.
  * Investigate hardware accelerated thumbnail generation.
  * Switch to Webp for thumbnails.
  * Test thumbnails laoading.
* Add observable status summary for each user - but build it with graphviz project that bakes to svg so it doesn't use live scripts.

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

## Design

TODO XXX diagram of connections ssh, browser, backend, authentication etc.

The code is organised in the following areas:
* [main.go](main.go) - the primary server.
* [static/browse.html](static/browse.html) - the main frontend browser page.

No frontend framework is used because adopting one on top of the simple interface design would have introduced unnecessary complexity.

This design for this solution favors simplicity and minimalism, both inside and out, without losing powerful features. *Filet Cloud Web* pushes a personal cloud solution to its leanest essence. It leaves you fully in control of your own data. It is a joy to use because it does what it needs to, reliably and quickly, and then gets out of the way. The primary design philosophy for this project is: **"complexity must justify itself, ruthlessly"**.

## Security

Since this service proxies SSH credentials, and both serves and modifies personal data, strict security policies have been implemented. Please use a modern and up-to-date browser and device to make full use of these protections.

Disclaimer: Use at your own risk. The codebase is strikingly small and the dependencies few, so the aim is that a security audit, for whosoever whishes to do it, should be as easy as possible. Nothing is secure until it is audited and reviewed by peers.

### Transport Security
* The login form will not open unless the connection protocol is HTTPS.
* HTTP requests to the main page are redirected to HTTPS.
* The backend implements a strict HTTPS ONLY policy. TODO (check for static and resources)
* HTTP Strict Transport Security (HSTS) is enforced. TODO
* All WebSocket connections use WebSocket Secure (WSS). TODO
* The Content Security Policy is configured to ensure that content is only loaded via HTTPS. TODO add https: to all parts of the content security policy.
* The backend supports being provided TLS credentials otherwise it uses an included certbot integration. TODO
* The webserver connects to the SFTP/SSH server without verifying the ssh host key so the connection between the filet-cloud-web server and the SFTP server cannot run across an untrusted network. This project intends for the SFTP server to be on the webserver localhost itself. Connecting to localhost is hardcoded to ensure this is the case. If you change this, ensure the HostKeyCallback is changed to use something secure.

### Authentication
* Authentication is made by proxying the SSH authentication mechanism through the backend in establishing a local SSH connection managed by the backend.
* This primarily relies on SSH username and password authentication.
* 2FA can be additionally configured with a Pluggable Authentication Module (PAM). TODO https://www.digitalocean.com/community/tutorials/how-to-set-up-multi-factor-authentication-for-ssh-on-ubuntu-20-04

### Login Session Managment
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
* Automatic logout will occur 5 minutes after a page remains not visible, such as after navigating to a new page, switching tabs, minimising the browser, or, on mobile, switching to another app. TODO https://developer.mozilla.org/en-US/docs/Web/API/Document/visibilitychange_event
* The user may choose to store the credentials in the browser's password management system, if supported and enabled in the browser. For additial security, 2FA is recommended. TODO

### Site Isolation and Content Protection
* Same-Origin Policy is enforced. TODO (https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy)
* Content Security Policy is configured to only allow content coming from the site's own origin. TODO (https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
* Cross Origin Isolation is enforced by: TODO
  * Setting the Cross Origin Opener Policy to ensure the browsing context is exclusively isolated to same-origin documents. TODO: All: Cross-Origin-Opener-Policy: same-origin
  * Setting the Cross Origin Embedder Policy to require corp (Cross Origin Resource Policy). TODO
  * Ensuring Cross Origin Isolation is fully activated by checking that the crossOriginIsolated property in the browser is active, before opening the login form. TODO  (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cross-Origin-Embedder-Policy To check if cross origin isolation has been successful, you can test against the crossOriginIsolated property available to window and worker contexts: )
* Default Cross Origin Read Blocking browser protections are enhanced by all Content Type Options being configured with nosniff, and the Content-Type header being set based on inspection of the first block. TODO X-Content-Type-Options: nosniff
* Cross Origin Resource Policy is configured to same-origin so that all resources are protected from access by any other origin. TODO ALL: Cross-Origin-Resource-Policy: same-origin
* Content Security Policy is enforced with a configuration that ensures: TODO (https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
  * Image and media content can only be loaded from the site's own origin. TODO
  * Script, stylesheet, and font resources can only be loaded from the site's own origin. TODO
  * Contents that do not match the above types, are denied. TODO
  * All content is loaded sandboxed with restricted allowances. TODO
  * Documents are prevented from being embedded. TODO
  * Forms are denied from using URLs as the target of form submission. TODO
  ``` browse.html (to be renamed to main.html)
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

### Authorised Browser Access to Content
* Along with storage requests being served via the authenticated WebSocket connection, authorised access is extended to the browser to allow the display of media and content, and for downloads. TODO
* Extension of authorised access to the browser is achieved via the following process: TODO
  * On login, the Storage class will:
    * Use the authenticated WebSocket connection to request an authentication JSON Web Token (JWT), which is created by the backend with the following JWT payload: TODO https://datatracker.ietf.org/doc/html/rfc7519#section-4.1
      * The remote address IP (which is never stored persistently by the backend) of the authenticated WebSocket connection's client side, as the Registered Audience Claim (aud). TODO
      * A time of 5 minutes later, as the Registered Expiration Time Claim (exp). TODO
      * A sequential identifier uniquely associated with the authenticated WebSocket connection, as the Registered Subject Claim (sub). TODO
    * Prevent exposure of the authorization JWT to JavaScript contexts outside of the Storage class. TODO
    * Send the JWT, via request body, to the backend's /authenticate endpoint so it can instruct the browser to set the JWT as an authentication cookie with the following cookie attribute protections: TODO https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-User
       * The browser only uses the authentication cookie in requests back to the site's own originating site, by setting SameSite=Strict. TODO
       * The browser expires the cookie after 5 minutes, by setting Max-Age=300. TODO
       * The authorization JWT is protected from JavaScript access, by setting HttpOnly. TODO
       * The cookie is further protected by setting the Secure cookie attribute, and by giving the cookie name to have the secure __Host- prefix. TODO
    * While the login session remains active, the Storage Class will keep the JWT refreshed by repeating this process on intervals. TODO
  * Requests to any storage link will succeed only if the backend's checks of the authentication JWT cookie is successfully validated with the following policy: TODO
    * The JWT is correctly signed. TODO
    * The remote address IP of the request must match the JWT's Registered Audience Claim. TODO
    * The JWT's Registered Expiration Time Claim must not have expired. TODO
    * The JWT's Registered Subject Claim (sub) must exactly match a unique identifier associated with an authenticated WebSocket connection, which is then used to fulfill the storage request. TODO
  * The JWT is signed using HS512 with a crytographically secure pseudorandom key generated on launch of the server TODO.

### Additional Cross-Site Request Forgery (CSRF/XSRF) Protection
* All backend endpoints which cause any changes or side effects (besides server load and establishing authentication), are only accessible through the WebSocket connection. TODO
* The WebSocket connection is stored in a private variable, inside the Storage class, and is only accessible via it's restricted API.
* No cross-site queries have access to that WebSocket connection.

### Third-Party Dependencies
* All third-party dependencies are servered from the backend and are version controlled and stored locally.
* All third-party dependencies loaded in the browser are Subresource Integrity checked. TODO

## Installation
* Ensure your machine allows ssh from localhost.
* Setup a certficate for TLS and ensure your browser respects it.
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
FC_CERT_FILE=my.crt FC_KEY_FILE=my.key ./filet-cloud-web
```
* Open in browser: `https://localhost/`

## Launch Options

Supported environment variables:
* `FC_LISTEN`: The address to listen on. Defaults to ':8443'.
* `FC_DOMAIN`: The domain to use with the included Let's Encrypt integration. Use of this implies acceptance of the LetsEncrypt Terms of Service.
* `FC_CERT_FILE` & `FC_KEY_FILE`: The cerdentials to use for TLS connections.
* `FC_DIR`: The folder path to use when serving storage, rather than the root. Supports a USERNAME token to serve a different tree for each user.
* `FC_SSH_PORT`: The port to use to connect locally.

## Development Testing

To set up TLS you could use a Self Signed Certificate with tools such as:
```bash
openssl req -x509 -newkey rsa:4096 -sha256 -days 1 -nodes -keyout my.key -out my.crt -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
openssl pkcs12 -export -in my.crt -inkey my.key -out my.p12
```

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
