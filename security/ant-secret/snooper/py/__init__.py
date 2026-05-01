try:
    # Python 3.x
    from builtins import bytes
except ImportError:
    # Python 2.x
    from __builtin__ import bytes

try:
    # arcadia python
    from ant_secret_snooper_bindings import Searcher as NativeSearcher
except ImportError:
    # system python
    from .ant_secret_snooper_bindings import Searcher as NativeSearcher


class Preset(object):
    # will choose some reasonable default
    DEFAULT = 0
    # only "known" secrets, e.g. OAuth-tokens, TVM-tickets, etc
    TRULY_SECRETS = 1
    # any secrets, like url-passwords, JWT-tokens, etc
    ALL = 2
    # some additions like MDS signs
    EXTRA = 3


class Snooper(object):
    __slots__ = ()

    def searcher(self, preset=Preset.ALL):
        return Searcher(preset)


class Searcher(object):
    def __init__(self, preset):
        self.searcher = NativeSearcher(preset=preset)

    def search(self, content, valid_only=False):
        secrets = self.searcher.search(
            data=content,
            valid_only=valid_only
        )

        return [
            Secret(
                type=SecretTypes._factory(secret["type"]),
                secret=secret["secret"],
                secret_from=secret["secret_from"],
                secret_len=secret["secret_len"],
                mask_from=secret["mask_from"],
                mask_len=secret["mask_len"],
            )
            for secret in secrets
        ]

    def mask(self, content, valid_only=False):
        secrets = self.search(content, valid_only)
        if not secrets:
            return content

        return mask_secrets(content, secrets)


class SecretType(object):
    def __init__(self, s_type, s_name):
        self.type = s_type
        self.name = s_name

    def __repr__(self):
        return "SecretType(%r, %r)" % (self.type, self.name)

    def __str__(self):
        return self.name


class SecretTypes(object):
    unknown = SecretType(0, "unknown")
    yandex_oauth = SecretType(1 << 0, "yandex_oauth")
    yandex_session = SecretType(1 << 1, "yandex_session")
    tvm_ticket = SecretType(1 << 2, "tvm_ticket")
    s3_presign = SecretType(1 << 3, "s3_presign")
    mds_sign = SecretType(1 << 4, "mds_sign")
    yc_api_key = SecretType(1 << 5, "yc_api_key")
    yc_cookie = SecretType(1 << 6, "yc_cookie")
    yc_token = SecretType(1 << 7, "yc_token")
    yc_static_cred = SecretType(1 << 8, "yc_static_cred")
    jwt_token = SecretType(1 << 9, "jwt_token")
    gitlab_token = SecretType(1 << 10, "gitlab_token")
    yrobo_password = SecretType(1 << 11, "yrobo_password")
    yhomo_password = SecretType(1 << 12, "yhomo_password")
    url_password = SecretType(1 << 13, "url_password")
    market_pull = SecretType(1 << 14, "market_pull")
    ms_cookie = SecretType(1 << 15, "ms_cookie")
    dev_api_key = SecretType(1 << 16, "dev_api_key")
    boxberry_token = SecretType(1 << 17, "boxberry_token")
    geosharing_token = SecretType(1 << 18, "geosharing_token")
    tvm_secret = SecretType(1 << 19, "tvm_secret")
    s3_secret_key = SecretType(1 << 20, "s3_secret_key")
    yf_token = SecretType(1 << 21, "yf_token")
    yc_refresh_token = SecretType(1 << 22, "yc_refresh_token")
    yc_oauth_secret = SecretType(1 << 23, "yc_oauth_secret")

    @staticmethod
    def _factory(req_type):
        for s_name, s_type in SecretTypes.__dict__.items():
            if isinstance(s_type, SecretType) and req_type == s_type.type:
                return getattr(SecretTypes, s_name)
        raise Exception("unknown secret type: %r" % req_type)


class Secret(object):
    __slots__ = ("type", "secret", "secret_from", "secret_to", "secret_len", "mask_from", "mask_to", "mask_len")

    def __init__(self, type, secret, secret_from, secret_len, mask_from, mask_len):
        self.type = type
        self.secret = secret
        self.secret_from = secret_from
        self.secret_to = self.secret_from + secret_len
        self.secret_len = secret_len
        self.mask_from = mask_from
        self.mask_to = self.mask_from + mask_len
        self.mask_len = mask_len

    def to_dict(self):
        return dict(
            type=self.type,
            secret=self.secret,
            secret_pos=self.secret_pos,
            mask_pos=self.mask_pos,
        )

    @property
    def mask_pos(self):
        return self.mask_from, self.mask_to

    @property
    def secret_pos(self):
        return self.secret_from, self.secret_to

    @property
    def is_yandex_oauth(self):
        return self.type == SecretTypes.yandex_oauth

    @property
    def is_yandex_session(self):
        return self.type == SecretTypes.yandex_session

    @property
    def is_tvm_ticket(self):
        return self.type == SecretTypes.tvm_ticket

    @property
    def is_s3_presign(self):
        return self.type == SecretTypes.s3_presign

    @property
    def is_mds_sign(self):
        return self.type == SecretTypes.mds_sign

    @property
    def is_yc_api_key(self):
        return self.type == SecretTypes.yc_api_key

    @property
    def is_yc_cookie(self):
        return self.type == SecretTypes.yc_cookie

    @property
    def is_yc_token(self):
        return self.type == SecretTypes.yc_token

    @property
    def is_yc_refresh_token(self):
        return self.type == SecretTypes.yc_refresh_token

    @property
    def is_yc_static_cred(self):
        return self.type == SecretTypes.yc_static_cred

    @property
    def is_yc_oauth_secret(self):
        return self.type == SecretTypes.yc_oauth_secret

    @property
    def is_s3_secret_key(self):
        return self.type == SecretTypes.s3_secret_key

    @property
    def is_jwt_token(self):
        return self.type == SecretTypes.jwt_token

    @property
    def is_gitlab_token(self):
        return self.type == SecretTypes.gitlab_token

    @property
    def is_yrobo_password(self):
        return self.type == SecretTypes.yrobo_password

    @property
    def is_yhomo_password(self):
        return self.type == SecretTypes.yhomo_password

    @property
    def is_url_password(self):
        return self.type == SecretTypes.url_password

    @property
    def is_market_pull(self):
        return self.type == SecretTypes.market_pull

    @property
    def is_ms_cookie(self):
        return self.type == SecretTypes.ms_cookie

    @property
    def is_dev_api_key(self):
        return self.type == SecretTypes.dev_api_key

    @property
    def is_boxberry_token(self):
        return self.type == SecretTypes.boxberry_token

    @property
    def is_geosharing_token(self):
        return self.type == SecretTypes.geosharing_token

    @property
    def is_tvm_secret(self):
        return self.type == SecretTypes.tvm_secret


def mask_secrets(data, secrets):
    must_decode = False
    if not isinstance(data, bytes):
        must_decode = True
        data = data.encode("utf-8")

    masked = bytearray(data)
    for secret in secrets:
        if secret.is_dev_api_key and secret.mask_len == len("****-****-****"):
            masked[secret.mask_from:secret.mask_to] = b"****-****-****"
            continue

        masked[secret.mask_from:secret.mask_to] = [0x58] * secret.mask_len

    result = bytes(masked)
    if must_decode:
        result = result.decode("utf-8")

    if hasattr(result, "__native__"):
        return result.__native__()
    return result
