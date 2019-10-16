# DNS Test Server

A DNS server suitable for testing. Records can be updated via HTTP requests.  Missing records are
resolved using the system resolver. This is distinct in behaviour from [Let's Encrypt's
challtestsrv](https://github.com/letsencrypt/challtestsrv) which has a notion of a default response,
which would not be suitable for general DNS usage.

The following are Examples of HTTP requests that the server supports:

    GET /
    GET /v1/a/www.google.com
    PUT /v1/a/www.google.com?v=127.0.0.1
    DELETE /v1/a/www.google.com
