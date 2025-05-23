import type { TextShortener } from './TextShortener';


const GO_VERSION_REGEX = /^(.*?)\/v(?:[2-9]|[1-9][0-9]+)([./].*)$/;
const GO_REGEX = /^(?:[\w-.]+\/)+([^.]+\..+).*$/;


const shorten = (text: string): Optional<string> => {
    let result = text;
    result = result.replace(GO_VERSION_REGEX, '$1$2');
    result = result.match(GO_REGEX)?.[1] ?? result;
    return result;
};

const testCases = [
    {
        input: 'syscall.Syscall',
        expected: 'syscall.Syscall',
    },
    {
        input: 'net/http.(*conn).serve',
        expected: 'http.(*conn).serve',
    },
    {
        input: 'github.com/blahBlah/foo.Foo',
        expected: 'foo.Foo',
    },
    {
        input: 'github.com/BlahBlah/foo.Foo',
        expected: 'foo.Foo',
    },
    {
        input: 'github.com/BlahBlah/foo.Foo[...]',
        expected: 'foo.Foo[...]',
    },
    {
        input: 'github.com/blah-blah/foo_bar.(*FooBar).Foo',
        expected: 'foo_bar.(*FooBar).Foo',
    },
    {
        input: 'encoding/json.(*structEncoder).(encoding/json.encode)-fm',
        expected: 'json.(*structEncoder).(encoding/json.encode)-fm',
    },
    {
        input: 'github.com/blah/blah/vendor/gopkg.in/redis.v3.(*baseClient).(github.com/blah/blah/vendor/gopkg.in/redis.v3.process)-fm',
        expected: 'redis.v3.(*baseClient).(github.com/blah/blah/vendor/gopkg.in/redis.v3.process)-fm',
    },
    {
        input: 'github.com/foo/bar/v4.(*Foo).Bar',
        expected: 'bar.(*Foo).Bar',
    },
    {
        input: 'github.com/foo/bar/v4/baz.Foo.Bar',
        expected: 'baz.Foo.Bar',
    },
    {
        input: 'github.com/foo/bar/v123.(*Foo).Bar',
        expected: 'bar.(*Foo).Bar',
    },
    {
        input: 'github.com/foobar/v0.(*Foo).Bar',
        expected: 'v0.(*Foo).Bar',
    },
    {
        input: 'github.com/foobar/v1.(*Foo).Bar',
        expected: 'v1.(*Foo).Bar',
    },
    {
        input: 'github.com/ClickHouse/clickhouse-go/v2.contextWatchdog.func1',
        expected: 'clickhouse-go.contextWatchdog.func1',
    },
    {
        input: 'example.org/v2xyz.Foo',
        expected: 'v2xyz.Foo',
    },
    {
        input: 'github.com/foo/bar/v4/v4.(*Foo).Bar',
        expected: 'v4.(*Foo).Bar',
    },
    {
        input: 'github.com/foo/bar/v4/foo/bar/v4.(*Foo).Bar',
        expected: 'v4.(*Foo).Bar',
    },
    {
        input: 'foo/xyz',
        expected: 'foo/xyz',
    },
];

export const go: TextShortener = {
    shorten,
    testCases,
};
