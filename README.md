# TIP - Token Introspection Proxy

Remote Token Introspection is just a TIP away.


TIP implements [AARC G052](https://aarc-community.org/guidelines/aarc-g052/). The focus of TIP is to enable OpenID 
Providers and OAuth Authorisation Servers that already implement a native token introspection endpoint per
[RFC7662](https://datatracker.ietf.org/doc/html/rfc7662) to support
[AARC G052](https://aarc-community.org/guidelines/aarc-g052/) without the need of additional implementations.

## How to TIP?

## How to deploy TIP

- TIP is deployed close to the existing AS.
- The existing [RFC7662](https://datatracker.ietf.org/doc/html/rfc7662) introspection endpoint is removed from the 
  metadata discovery.
- The introspection endpoint provided by TIP is added as `introspection_endpoint` to the metadata discovery.

## What does TIP do?

0. TIP receives token introspection requests.
0. TIP inspects the token in the request and determines the issuer of the token:
    - If the issuer is the linked AS, the request is replayed at the
      [RFC7662](https://datatracker.ietf.org/doc/html/rfc7662) introspection endpoint and the response is forwarded 
      without modifications to the client.
    - Otherwise, the client authentication must be checked.
0. To check client authentication the only option currently implemented requires TIP to send a dummy introspection 
   request to the [RFC7662](https://datatracker.ietf.org/doc/html/rfc7662) introspection endpoint of the linked AS 
   including a dummy token, but the real authentication from the client.
    - If the client authentication fails, TIP will return a `401` error.
    - If the client authentication is successful, TIP continues the (remote) introspection
0. TIP continues to evaluate the issuer of the token:
    - If the issuer cannot be determined, the token might be sent to a fallback issuer for remote introspection, 
      otherwise `{"active": false"}` is returned.
    - If the issuer can be determined, but is not supported, the token might be sent to a fallback issuer for remote 
      introspection, otherwise `{"active": false"}` is returned.
    - If the issuer is supported, the token is sent to the issuer's token introspection endpoint for remote 
      introspection.
0. If the response from the remote introspection is negative, TIP returns `{"active": false"}`
0. If the response is positive, TIP can do some translation and renaming of claims before returning the response to 
   the client.

## Configuration
For an example configuration (including comments) please see [example-config.yaml](example-config.yaml).

## Future Work

- Support for OpenID Federation will be added. Then it is not required to register a client with remote issuers (as 
  long as they are part of the same federation).
- ...