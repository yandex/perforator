Library to get access for individual struct members.
Library is inspired by [apolukhin/magic_get](https://github.com/apolukhin/magic_get)

#### Usage

```
    struct TSample {
        int field1;
        int field2;
    };

    TSample sample{1, 2};
    size_t membersCount = NIntrospection::MembersCount<TSample>();  // membersCount == 2
    const auto members = NIntrospection::Members<TSample>();  // get const reference to all members as std::tuple<const int&, const int&>
    const auto&& field1 = NIntrospection::Member<0>(sample);  // const reference to TSample::field1
    const auto&& field2 = NIntrospection::Member<1>(sample);  // const reference to TSample::field2
```

#### Limitiations:

* Provides read-only access to members (cannot be used for modification)
* Supported only Only C-style structures w/o user-declared or inherited constructors (see https://en.cppreference.com/w/cpp/language/aggregate_initialization)
* C-style array members are not supported.

Example:

```
struct TSample {
   int a[3];
}
```

the library determines that the structure has 3 members instead of 1.
As workarraund the std::array can be used:

```
struct TSample {
   std::array<int, 3> a;
}
```

