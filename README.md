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

```mermaid
flowchart TD
   A[TIP receives token introspection request]
   AA[TIP inspects the token in the request and determines the issuer of the token]
   B{Is issuer the linked AS?}
   C[Create new request using parameters and credentials from original request]
   CC[Send request to linked AS's RFC7662 endpoint]
   D[Return response to client unmodified]
   E[Check client authentication]
   F[Send dummy request with original client credentials
    but dummy token
    to linked AS's RFC7662 endpoint]
   G{Client auth valid?}
   H[Return 401 Unauthorized]
   I[Continue remote introspection]
   J{Can issuer
    be determined?}
   K{Is issuer supported?}
   L{Is there a
    fallback issuer
    configured?}
   Q[Send to fallback issuer's introspection endpoint]
   M[Send token to issuer's introspection endpoint]
   N{Response active?}
   O[Return active=false]
   P[Translate and rename claims
    according to configured rules]
   R[Return updated introspection response]

   A --> AA
   AA --> B
   B -- Yes --> C --> CC --> D
   B -- No --> E --> F --> G
   G -- No --> H
   G -- Yes --> I --> J
   K -- No --> L
   J -- No --> L
   J -- Yes --> K
   K -- Yes --> M --> N
   N -- No --> O
   Q --> N
   N -- Yes --> P
   L -- Yes --> Q
   L -- No --> O
   P --> R
```

## Configuration
For an example configuration (including comments) please see [example-config.yaml](example-config.yaml).

## Docker Image
The docker image [myoidc/tip](https://hub.docker.com/r/myoidc/tip) is available at
[dockerhub](https://hub.docker.com/r/myoidc/tip).

## Future Work

- Support for OpenID Federation will be added. Then it is not required to register a client with remote issuers (as 
  long as they are part of the same federation).
- ...