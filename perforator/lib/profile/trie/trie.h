#pragma once

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>

#include <util/generic/vector.h>

#include <concepts>

namespace NPerforator::NProfile {

// Generic trie with Struct-of-Arrays layout.
template <typename TKey, typename TValue, std::integral TIndex = ui32>
class TTrie {
private:
    struct TEdgeKey {
        TKey Key;
        TIndex ParentId;

        bool operator==(const TEdgeKey& other) const = default;

        template <typename H>
        friend H AbslHashValue(H h, const TEdgeKey& key) {
            return H::combine(std::move(h), key.ParentId, key.Key);
        }
    };

    // Most CCT nodes have tiny fanout (measured: ~90% have <=1 child, ~99%
    // have <=4). Storing the first few children inline lets those lookups avoid a
    // random probe into the global edge hash entirely; only high-degree nodes spill
    // to EdgeToNode_. Byte-identical to the pure-hash trie (same creation order).
    static constexpr ui32 InlineCap = 4;
    struct TChildSlots {
        // Unused slots are zero. A child id of 0 means "empty" — node 0 is the root,
        // which is never anyone's child — so no separate count field is needed. Slots
        // are filled in order, so the first zero terminates the occupied prefix.
        TKey Keys[InlineCap] = {};
        TIndex Children[InlineCap] = {};
    };

public:
    friend class TNode;

    class TNode {
    public:
        TNode(TTrie* trie, TIndex id)
            : Trie_{trie}
            , Id_{id}
        {}

        TIndex GetId() const {
            return Id_;
        }

        bool IsZero() const {
            return Id_ == 0;
        }

        const TKey& GetKey() const {
            return Trie_->Keys_[Id_];
        }

        const TValue& GetValue() const {
            return Trie_->Values_[Id_];
        }

        TValue& GetValue() {
            return Trie_->Values_[Id_];
        }

        TNode GetFirstChild() const {
            return {Trie_, Trie_->FirstChild_[Id_]};
        }

        TNode GetNextSibling() const {
            return {Trie_, Trie_->NextSibling_[Id_]};
        }

        TNode GetOrCreateChild(const TKey& key) {
            // Inline fast path: scan this node's few inline children (cache-local — the
            // parent was just touched). Only high-degree nodes spill to the global hash.
            const TChildSlots& slots = Trie_->Inline_[Id_];
            for (ui32 i = 0; i < InlineCap; ++i) {
                if (slots.Children[i] == 0) {
                    // Empty slot (and everything after): key absent — insert inline here.
                    TIndex newId = Trie_->CreateNode(key, Id_);
                    TChildSlots& s = Trie_->Inline_[Id_];  // re-fetch: CreateNode may realloc
                    s.Keys[i] = key;
                    s.Children[i] = newId;
                    return {Trie_, newId};
                }
                if (slots.Keys[i] == key) {
                    return {Trie_, slots.Children[i]};
                }
            }

            TEdgeKey edgeKey{key, Id_};
            auto [it, inserted] = Trie_->EdgeToNode_.try_emplace(edgeKey, 0);
            if (!inserted) {
                return {Trie_, it->second};
            }
            it->second = Trie_->CreateNode(key, Id_);  // CreateNode doesn't touch EdgeToNode_
            return {Trie_, it->second};
        }

    private:
        TTrie* Trie_;
        TIndex Id_;
    };

    TTrie()
        : Keys_{{}}
        , Values_{{}}
        , FirstChild_{0}
        , NextSibling_{0}
        , Parent_{0}
        , Inline_(1)
    {}

    TNode Root() {
        return {this, 0};
    }

    TNode Root() const {
        return {const_cast<TTrie*>(this), 0};
    }

    TNode NodeAt(TIndex idx) {
        return {this, idx};
    }

    TNode NodeAt(TIndex idx) const {
        return {const_cast<TTrie*>(this), idx};
    }

    TIndex NodeCount() const {
        return Keys_.size();
    }

    void Finalize() {
        decltype(EdgeToNode_){}.swap(EdgeToNode_);
        decltype(Inline_){}.swap(Inline_);  // build-only; render walks the sibling list
    }

    // Post-order reduction toward the root: combine each non-root node's value into its
    // parent's, deepest first. A node's id always exceeds its parent's, so a reverse-id
    // sweep is a valid post-order. The caller supplies the combine, so TTrie imposes
    // nothing on TValue.
    template <typename TCombine>
    void ReduceToRoot(TCombine combine) {
        for (TIndex i = Keys_.size(); i-- > 1; ) {
            combine(Values_[Parent_[i]], Values_[i]);
        }
    }

private:
    // Append a node as a child of `parent`, prepending it to parent's sibling list.
    // Does not touch EdgeToNode_ or the parent's inline slots (callers handle indexing).
    TIndex CreateNode(const TKey& key, TIndex parent) {
        TIndex newId = Keys_.size();
        Keys_.push_back(key);
        Values_.push_back({});
        FirstChild_.push_back(0);
        NextSibling_.push_back(FirstChild_[parent]);
        Parent_.push_back(parent);
        FirstChild_[parent] = newId;
        Inline_.push_back(TChildSlots{});
        return newId;
    }

    TVector<TKey> Keys_;
    TVector<TValue> Values_;
    TVector<TIndex> FirstChild_;
    TVector<TIndex> NextSibling_;
    TVector<TIndex> Parent_;
    TVector<TChildSlots> Inline_;
    absl::flat_hash_map<TEdgeKey, TIndex> EdgeToNode_;
};

} // namespace NPerforator::NProfile
