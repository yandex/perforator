GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.30.0)

SRCS(
    deviceauth.go
    oauth2.go
    pkce.go
    token.go
    transport.go
)

END()

RECURSE(
    amazon
    authhandler
    bitbucket
    cern
    clientcredentials
    endpoints
    facebook
    fitbit
    foursquare
    github
    gitlab
    google
    heroku
    hipchat
    instagram
    internal
    jira
    jws
    jwt
    kakao
    linkedin
    mailchimp
    mailru
    mediamath
    microsoft
    nokiahealth
    odnoklassniki
    paypal
    slack
    spotify
    stackoverflow
    twitch
    uber
    vk
    yahoo
    yandex
)
