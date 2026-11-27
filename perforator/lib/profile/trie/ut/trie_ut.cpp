#include <perforator/lib/profile/trie/trie.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <memory>

using namespace NPerforator::NProfile;

TEST(TTrie, Basic) {
    TTrie<int, i64> trie;

    EXPECT_EQ(trie.NodeCount(), 1u);
    EXPECT_TRUE(trie.Root().GetFirstChild().IsZero());
}

TEST(TTrie, AddChildren) {
    TTrie<int, i64> trie;

    auto root = trie.Root();
    auto child1 = root.GetOrCreateChild(1);
    auto child2 = root.GetOrCreateChild(2);
    auto child3 = root.GetOrCreateChild(3);

    EXPECT_EQ(trie.NodeCount(), 4u);
    EXPECT_EQ(child1.GetKey(), 1);
    EXPECT_EQ(child2.GetKey(), 2);
    EXPECT_EQ(child3.GetKey(), 3);
}

TEST(TTrie, GetOrCreateIsIdempotent) {
    TTrie<int, i64> trie;

    auto root = trie.Root();
    auto first = root.GetOrCreateChild(42);
    auto second = root.GetOrCreateChild(42);

    EXPECT_EQ(first.GetId(), second.GetId());
    EXPECT_EQ(trie.NodeCount(), 2u);
}

TEST(TTrie, DeepTree) {
    TTrie<int, i64> trie;

    auto node = trie.Root();
    for (int i = 1; i <= 100; ++i) {
        node = node.GetOrCreateChild(i);
    }

    EXPECT_EQ(trie.NodeCount(), 101u);
    EXPECT_EQ(node.GetKey(), 100);
}

TEST(TTrie, AddValue) {
    TTrie<int, i64> trie;

    auto root = trie.Root();
    auto child = root.GetOrCreateChild(1);
    child.GetValue() += 10;
    child.GetValue() += 5;

    EXPECT_EQ(child.GetValue(), 15);
    EXPECT_EQ(root.GetValue(), 0);
}

TEST(TTrie, SiblingTraversal) {
    TTrie<int, i64> trie;

    auto root = trie.Root();
    root.GetOrCreateChild(1);
    root.GetOrCreateChild(2);
    root.GetOrCreateChild(3);

    // Collect all children via sibling links
    TVector<int> children;
    for (auto child = root.GetFirstChild(); !child.IsZero(); child = child.GetNextSibling()) {
        children.push_back(child.GetKey());
    }

    EXPECT_EQ(children.size(), 3u);
    // Children are added in reverse order (newest first)
    EXPECT_EQ(children[0], 3);
    EXPECT_EQ(children[1], 2);
    EXPECT_EQ(children[2], 1);
}

TEST(TTrie, Finalize) {
    TTrie<int, i64> trie;

    auto root = trie.Root();
    root.GetOrCreateChild(1);
    root.GetOrCreateChild(2);
    trie.Finalize();

    // After finalize, structure should still be accessible
    EXPECT_EQ(trie.NodeCount(), 3u);
    EXPECT_EQ(root.GetFirstChild().GetId(), 2u);
}

namespace {

struct alignas(64) TKey {
    TKey() = delete;

    explicit TKey(int value)
        : Value(std::make_shared<int>(value))
    {}

    bool operator==(const TKey& other) const {
        return *Value == *other.Value;
    }

    template <typename H>
    friend H AbslHashValue(H hash, const TKey& key) {
        return H::combine(std::move(hash), *key.Value);
    }

    std::shared_ptr<int> Value;
};

}

TEST(TTrie, NonDefaultKeysAndObjectLifetimes) {
    TVector<std::weak_ptr<int>> values;
    {
        TKey rootKey{-1};
        TTrie<TKey, i64> trie{rootKey};
        EXPECT_EQ(rootKey.Value.use_count(), 2);
        values.push_back(rootKey.Value);
        for (int i = 0; i < 128; ++i) {
            TKey key{i};
            auto child = trie.Root().GetOrCreateChild(key);
            child.GetValue() = i;
            EXPECT_EQ(child.GetId(), trie.Root().GetOrCreateChild(TKey{i}).GetId());
            EXPECT_EQ(reinterpret_cast<uintptr_t>(&child.GetKey()) % alignof(TKey), 0u);
            values.push_back(key.Value);
        }
        auto copy = trie;
        auto moved = std::move(copy);
        moved.Root().GetOrCreateChild(TKey{128});
        EXPECT_EQ(trie.NodeCount(), 129u);
        EXPECT_EQ(moved.NodeCount(), 130u);
        trie.Finalize();
        moved.Finalize();
        int count = 0;
        for (auto child = trie.Root().GetFirstChild(); !child.IsZero(); child = child.GetNextSibling()) {
            EXPECT_EQ(child.GetValue(), *child.GetKey().Value);
            ++count;
        }
        EXPECT_EQ(count, 128);
        for (const auto& value : values) {
            EXPECT_FALSE(value.expired());
        }
    }
    for (const auto& value : values) {
        EXPECT_TRUE(value.expired());
    }
}

TEST(TTrie, InsertsUsingKeyFromOwnStorage) {
    TTrie<TKey, i64> trie{TKey{42}};
    auto node = trie.Root();
    for (int i = 0; i < 1024; ++i) {
        node = node.GetOrCreateChild(trie.Root().GetKey());
        EXPECT_EQ(*node.GetKey().Value, 42);
    }
    EXPECT_EQ(trie.NodeCount(), 1025u);
}
