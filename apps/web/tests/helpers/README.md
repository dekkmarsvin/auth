# Account-page test support

These small helpers were moved unchanged from the previously bundled Web Kit
`9285460e` tests. They compile this host's actual App SFC and exercise Vue events
with a minimal renderer; they do not implement or patch the SDK. Keeping them
here removes the account-flow regression's dependency on an SDK source checkout.
