import re

from collections import namedtuple
import hashlib

import devtools.ya.test.const as test_const

SUPPORTED_INTEGRITY_ALG = {'md5', 'sha1', 'sha512'}


class MissingResourceError(Exception):
    mute = True


class InvalidHttpUriException(Exception):
    def __init__(self, reason, uri):  # type: (str, str) -> None
        msg = 'Wrong uri. {0}: {1}'.format(reason, uri)
        super(InvalidHttpUriException, self).__init__(msg)


class InvalidUriSchemaException(Exception):
    def __init__(self, allowed_schemas, uri):  # type: (set[str], str) -> None
        msg = 'Unsupported URI schema: expected one of ({}), got {}'.format(', '.join(sorted(allowed_schemas)), uri)
        super(InvalidUriSchemaException, self).__init__(msg)


ParsedResourceUri = namedtuple(
    'ParsedResourceUri', 'resource_type, resource_uri, resource_id, resource_url, fetcher_meta'
)


def md5_hex(string):  # type: (str) -> str
    return hashlib.md5(string.encode()).hexdigest()


def validate_parsed_http_meta(meta, resource_uri):  # type: (dict[str,str], str) -> None
    integrity = meta.get('integrity', None)
    if not integrity:
        raise InvalidHttpUriException('Integrity is required but not provided', resource_uri)
    if '-' not in integrity:
        raise InvalidHttpUriException('Integrity should be in form {alg}-{data}', resource_uri)
    alg, _ = integrity.split('-', 1)
    if alg not in SUPPORTED_INTEGRITY_ALG:
        algorithms = ", ".join(sorted(SUPPORTED_INTEGRITY_ALG))
        raise InvalidHttpUriException(
            'Integrity algorithm supports {0}, got \'{1}\''.format(algorithms, alg), resource_uri
        )


def parse_http_uri_meta(meta_str, resource_uri):  # type: (str,str) -> tuple[dict[str, str], str]
    pairs = meta_str.split('&')
    meta_dict = {}  # type: dict[str,str]
    for pair in pairs:
        key, value = pair.split('=', 1)
        meta_dict[key] = value.replace('%26', '&')

    validate_parsed_http_meta(meta_dict, resource_uri)
    resource_id = md5_hex(meta_dict.get('integrity'))
    return meta_dict, resource_id


def parse_http_uri(resource_uri):  # type: (str) -> ParsedResourceUri
    if '#' not in resource_uri:
        raise InvalidHttpUriException('No \'#\' symbol is found', resource_uri)

    resource_url, meta_str = resource_uri.split('#', 1)

    if not meta_str:
        raise InvalidHttpUriException('\'#\' symbol must be followed by resource meta', resource_uri)

    if '=' not in meta_str:
        # backward compatibility with old md5-scheme
        return ParsedResourceUri('http', resource_uri, meta_str, resource_url, fetcher_meta=None)

    meta, resource_id = parse_http_uri_meta(meta_str, resource_uri)
    return ParsedResourceUri('http', resource_uri, resource_id, resource_url, fetcher_meta=meta)


def parse_resource_uri(resource_uri, force_accepted_schemas=None):  # type: (str, set[str]) -> ParsedResourceUri
    if not force_accepted_schemas:
        try:
            import app_ctx

            accepted_schemas = app_ctx.fetchers_storage.accepted_schemas()
        except (ImportError, AttributeError):  # internal tests can have no app_ctx or configured fetchers_storage
            accepted_schemas = {'sbr', 'http', 'https', 'docker'}
    else:
        accepted_schemas = force_accepted_schemas

    resource_type, rest = resource_uri.split(':', 1)
    if resource_type in ('http', 'https'):
        return parse_http_uri(resource_uri)
    elif resource_type == 'base64':
        return ParsedResourceUri(resource_type, resource_uri, resource_id=None, resource_url=rest, fetcher_meta=None)
    elif resource_type in accepted_schemas:
        if resource_type == 'docker':
            resource_id = get_docker_resource_id(resource_uri)
        else:
            resource_id = rest
        return ParsedResourceUri(resource_type, resource_uri, resource_id, resource_url=None, fetcher_meta=None)
    else:
        raise InvalidUriSchemaException(accepted_schemas, resource_uri)


def get_mapped_parsed_uri_and_info(parsed_uri, mapping, resource_info):
    # type: (ParsedResourceUri, dict, dict) -> tuple[ParsedResourceUri, dict]

    resource_id = parsed_uri.resource_id

    if resource_id not in mapping.get("resources", {}):
        raise MissingResourceError("Resource mapping doesn't have required resource: {}".format(resource_id))

    if resource_id in mapping.get("resources_info", []):
        resource_info = mapping["resources_info"][resource_id]

    new_url = mapping["resources"][resource_id]

    new_parsed_uri = ParsedResourceUri(
        resource_type=parsed_uri.resource_type,
        resource_uri=parsed_uri.resource_uri,
        resource_id=parsed_uri.resource_id,
        resource_url=new_url,
        fetcher_meta=parsed_uri.fetcher_meta,
    )

    return new_parsed_uri, resource_info


def get_docker_resource_id(resource_uri):
    res = re.match(test_const.DOCKER_LINK_RE, resource_uri)
    fqdn = res.group(2)
    digest = res.group(4)
    digest_prefix_len = 12
    return fqdn + '-' + digest[:digest_prefix_len]
