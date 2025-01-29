import type { TextShortener } from './TextShortener';


const makeRegex = (): RegExp => {
    const ids = '[_A-z]\\w*';
    const classTemplateParameters = '(?:<[^>]*>)?';
    const functionTemplateParameters = '(?:<.*>)?';
    const namespaces = `${ids}${classTemplateParameters}::`;
    const destructor = '~?';
    const functions = `${destructor}${ids}${functionTemplateParameters}`;
    const regexString = `^(?:${namespaces})*((?:${namespaces})${functions}).*$`;
    return new RegExp(regexString);
};

const CPP_REGEX = makeRegex();

const shorten = (text: string): Optional<string> => {
    let result = text;
    result = result.replace(/\(anonymous namespace\)::/g, '');
    result = result.match(CPP_REGEX)?.slice(1).join('') ?? result;
    return result;
};

const testCases = [
    {
        input: 'name_space::class::method',
        expected: 'class::method',
    },
    {
        input: 'namespace1::namespace2::class::method',
        expected: 'class::method',
    },
    {
        input: '(anonymous namespace)::Class::Method',
        expected: 'Class::Method',
    },
    {
        input: 'class::method',
        expected: 'class::method',
    },
    {
        input: '(anonymous namespace)::function',
        expected: 'function',
    },
    {
        input: 'namespace::Class::method()',
        expected: 'Class::method',
    },
    {
        input: 'namespace::Class::method(int, std::string)',
        expected: 'Class::method',
    },
    {
        input: 'class::method @/main.cpp',
        expected: 'class::method',
    },
    {
        input: 'namespace1::namespace2::Class::method<int>() @/main.cpp',
        expected: 'Class::method<int>',
    },
    {
        input: 'namespace::Class::method()::$_42::operator+',
        expected: 'Class::method',
    },
    {
        input: 'namespace::Class::~Class',
        expected: 'Class::~Class',
    },
    {
        input: 'namespace::Class<float>::method()',
        expected: 'Class<float>::method',
    },
];

export const cpp: TextShortener = {
    shorten,
    testCases,
};
