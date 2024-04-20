# Security Design
Since this service proxies SSH credentials, and both serves and modifies personal data, hardened security policies have been implemented. Please use a modern and up-to-date browser and device to make full use of these protections.

## Transport Security
* The login form will not open unless the connection protocol is HTTPS.
* HTTP requests to the main page are redirected to HTTPS.
* The backend implements a strict HTTPS ONLY policy.
* HTTP Strict Transport Security (HSTS) is enabled.
* All WebSocket connections use WebSocket Secure (WSS).
* The backend supports being provided TLS credentials otherwise it uses an included certbot integration.
* The webserver connects to the local SFTP/SSH service without verifying the SSH Host Key, therefore the connection between them cannot run across an untrusted network. Connecting to localhost is hardcoded to ensure this is the case. If you change this, ensure the HostKeyCallback is changed to use something secure, or reroute through a local connection using SSH port forwarding.

## Authentication
* Authentication is made by proxying the SSH credentials through the backend in establishing a local SSH connection managed by the backend.
* This primarily relies on SSH username and password authentication.
* 2FA can be additionally configured with a Pluggable Authentication Module (PAM).

## Login Session Managment
* On completion of the login form, an authenticated secure connection is established:
  * A WebSocket connection is established with the backend (with WSS).
  * The login credentials are passed directly to the WebSocket connection.
  * The backend passes the credentials directly into establishing an SFTP/SSH connection locally.
  * The SFTP/SSH connection is attached to the WebSocket connection to handle future requests.
  * The credentials are not stored in any persistent way.
  * Failure to establish an authenticated SFTP/SSH connection will close the WebSocket connection, triggering a new login sequence.
  * After sending the credentials to the WebSocket connection, the login form will pass the potentially authenticated WebSocket connection to be stored inside an instance of the Storage class in a private variable, so as to restrict direct access from JavaScript except via its API. Note that this is also shared with the terminal element.
* The user may choose to store the credentials in the browser's password management system, if supported and enabled in the browser. For additial security, 2FA is recommended.
* Logout occurs when either the browser or the backend closes the WebSocket connection, such as:
  * Automatically when closing or refreshing the browser tab.
  * When restarting the backend service.
  * From a disruption to the network connection.
* Logout closes the WebSocket connection, triggering the server to close its end and invalidate any established authentication cookies.
* Automatic logout will also occur 5 minutes after a page remains not visible, such as after navigating to a new page, switching tabs, minimising the browser, or, on mobile, switching to another app.
* Logging out via the logout button will additionally cleaer all site data, as supported by the browser.

## Authorised Browser Access to Content
* Along with storage requests being served via the authenticated WebSocket connection, authorised access is extended to the browser to allow the display of media and content, and for downloads.
* Extension of authorised access to the browser is achieved via the following process:
  * On login, the Storage class will:
    * Use the authenticated WebSocket connection to request an authentication JSON Web Token (JWT), which is created by the backend with the following JWT payload:
      * The remote address IP (which is never stored persistently by the backend) of the authenticated WebSocket connection's client side, as the Registered Audience Claim (aud).
      * A time of 5 minutes later, as the Registered Expiration Time Claim (exp).
      * A sequential identifier uniquely associated with the authenticated WebSocket connection, as the Registered Subject Claim (sub).
    * Prevent exposure of the authorization JWT to JavaScript contexts outside of the Storage class.
    * Send the JWT, via request body, to the backend's /authenticate endpoint so it can instruct the browser to set the JWT as an authentication cookie with the following cookie attribute protections:
       * The browser only uses the authentication cookie in requests back to the site's own originating site, by setting SameSite=Strict.
       * The browser expires the cookie after 5 minutes, by setting Max-Age=300.
       * The authorization JWT is protected from JavaScript access, by setting HttpOnly.
       * The cookie is further protected by setting the Secure cookie attribute, and by giving the cookie the secure `__Host-` prefix.
    * While the login session remains active, the Storage Class will keep the JWT refreshed by repeating this process on intervals.
  * Requests to any storage link (via the `/file:/` `/thumb:/` and `/zip` URL paths) will succeed only if the backend's checks of the authentication JWT cookie is successfully validated against the following policy:
    * The JWT is correctly signed.
    * The remote address IP of the request must match the JWT's Registered Audience Claim.
    * The JWT's Registered Expiration Time Claim must not have expired.
    * The JWT's Registered Subject Claim must exactly match a unique identifier associated with an authenticated WebSocket connection, which is then used to fulfill the storage request.
  * The JWT is signed using HS512 with a crytographically secure pseudorandom key generated on launch of the server.

## Site Isolation and Content Protection
* Same-Origin Policy is enforced.
* Cross Origin Isolation is enforced by:
  * Setting the Cross Origin Opener Policy to ensure the browsing context is exclusively isolated to same-origin documents.
  * Setting the Cross Origin Embedder Policy to require corp (Cross Origin Resource Policy).
  * Ensuring Cross Origin Isolation is fully activated by checking that the crossOriginIsolated property in the browser is active, before opening the login form.
  * Default Cross Origin Read Blocking browser protections are enhanced by all Content Type Options being configured with nosniff, and with the Content-Type header being set.
* Cross Origin Resource Policy is configured to same-origin so that all resources are protected from access by any other origin.
* Content Security Policy is enforced with a configuration that ensures:
  * Image, font and media content can be loaded only from the site's own origin.
  * Script resources can be loaded only from the site's own origin or from inline elements protected with a 128 bit cryptographically secure random nonce.
  * Stylesheet resources can be loaded only from the site's own origin or from inline elements.
  * WebSockets can only be connected to the site own origin.
  * Contents that do not match the above types, are denied.
  * All content is loaded sandboxed with restricted allowances.
  * Documents are prevented from being embedded.
  * Forms are denied from using URLs as the target of form submission.
* The backend requires the browser to provide Secure Fetch Metadata Request Headers, and denies access to content unless the following policies are met:
  * For the main page:
    * The request destination is a document, preventing embedding.
  * For the `/resources/` URL path:
    * The request site is same-origin.
    * The request destination is a script, style or font element.
  * For the `/preconnect` endpoint:
    * The request site is same-origin.
    * The request destination is set to the word empty.
  * For the `/connect` endpoint:
    * If any site, mode, or destination Secure Fetch Metadata Headers are provided, then they must all match the policy:
      * The request site is same-origin.
      * The request mode is a websocket.
      * The request destination is set to the word empty (Firefox) or websocket (Safari).
    * Unfortunately, Chromium based browsers do not send any Secure Fetch Metadata Headers (as of Chrome Version 123.0.6312.124) when establishing WebSocket connections. To ensure the /connect endpoint is still protected by a same-origin site check, this endpoint expects a __Host-SecSiteSameOrigin cookie to contain a valid JWT with valid expiration and audience claims, which can only be obtained from the /preconnect endpoint as a SameSite=Strict cookie after its own checks that the site is same origin. The JWT provided by the /preconnect endpoint is given an audience claim of the client IP, and an expiration claim of 3 seconds after the time of creation.
  * For the `/authenticate` endpoint:
    * The request site is same-origin.
    * The request destination is set to the word empty.
  * For `/file:/` `/thumb:/` and `/zip` URL paths:
    * If any site, mode, or destination Secure Fetch Metadata Headers are provided, then they must all match the policy:
      * The request site is same-origin.
      * The request destination is audio, an image, a video, a document, or is set to the word empty.
    * Unfortunately, Safari based browsers do not send any Secure Fetch Metadata Headers (as of Safari Version 17.4.1 (19618.1.15.11.14)) when downloading files. To ensure these endpoints are still protected by a same-origin site check, these endpoints each expect a __Host-SecSiteSameOrigin cookie to contain a valid JWT with valid expiration and audience claims, which can only be obtained from the /preconnect endpoint as a SameSite=Strict cookie after its own checks that the site is same origin. The JWT provided by the /preconnect endpoint is given an audience claim of the client IP, and an expiration claim of 3 seconds after the time of creation.
* The backend enforces a browser cache policy which ensures cached content access adheres to the above Secure Fetch Metadata Request Header policy, including when the headers vary across subsequent requests.
* A Referrer Policy of same-origin is enforced.

## Additional Cross-Site Request Forgery (CSRF/XSRF) Protection
* The login form is protected with a CSRF Token secured by an HMAC-SHA256 Signed Double-Submit Cookie.
* All backend endpoints which cause any changes or side effects (besides server load or establishing authentication), are only accessible through the WebSocket connection.
* The WebSocket connection is stored in a private variable, inside the Storage class, and is only accessible via it's restricted API.

## Third-Party Dependencies
* All third-party dependencies are servered from the backend and are version controlled and stored locally.
* All third-party dependencies loaded in the browser are Subresource Integrity checked.
* Cache-Control is enforced so the browser caches content for no longer than 10 hours.

