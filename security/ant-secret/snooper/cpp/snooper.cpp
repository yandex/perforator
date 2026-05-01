#include "snooper.h"

#include <security/ant-secret/snooper/internal/searcher.h>
#include <security/ant-secret/snooper/internal/searchers_store.h>
#include <security/ant-secret/internal/validation/validator.h>

#include <util/system/mutex.h>
#include <util/generic/singleton.h>

namespace NSnooper {
    namespace {
        class TGenericProducer {
            public:
                explicit TGenericProducer(NSnooperInt::TContext ctx)
                    : ctx(std::move(ctx))
                    {}

                NSnooperInt::TSearchersStore* Get(TSecretTypes neededSecrets) {
                    with_lock (lock) {
                        const auto it = stores.find(neededSecrets);
                        if (it != stores.end()) {
                            return it->second.Get();
                        }

                        auto&& [item, ok] = stores.emplace(std::piecewise_construct,
                            std::forward_as_tuple(neededSecrets),
                            std::forward_as_tuple(new NSnooperInt::TSearchersStore(ctx, neededSecrets)));

                        if (Y_UNLIKELY(!ok)) {
                            ythrow TSystemError() << "failed to searchers store";
                        }

                        return item->second.Get();
                    }
                }
            protected:
                NSnooperInt::TContext ctx;
                TMutex lock;
                THashMap<NSecret::TSecretTypes, THolder<NSnooperInt::TSearchersStore>> stores;
        };

        class TConcreteProducer {
            public:
                TConcreteProducer(NSnooperInt::TContext ctx, TSecretTypes neededSecrets) {
                    store = MakeHolder<NSnooperInt::TSearchersStore>(ctx, neededSecrets);
                }

                NSnooperInt::TSearchersStore* Get() {
                    return store.Get();
                }

            protected:
                THolder<NSnooperInt::TSearchersStore> store;
        };

        class TTrulySecretsProducer : public TConcreteProducer {
            public:
                explicit TTrulySecretsProducer(NSnooperInt::TContext ctx)
                    : TConcreteProducer(ctx, ESecretType::TrulySecrets) {}
        };

        class TAllSecretsProducer : public TConcreteProducer {
            public:
                explicit TAllSecretsProducer(NSnooperInt::TContext ctx)
                    : TConcreteProducer(ctx, ESecretType::All) {}
        };

        class TAllWMdsSecretsProducer : public TConcreteProducer {
            public:
                explicit TAllWMdsSecretsProducer(NSnooperInt::TContext ctx)
                    : TConcreteProducer(ctx, ESecretType::AllWMds) {}
        };

        NSnooperInt::TSearchersStore* searchersStore(NSnooperInt::TContext ctx, TSecretTypes neededSecrets) {
            switch (static_cast<ESecretType>(neededSecrets.ToBaseType())) {
                case ESecretType::TrulySecrets:
                    return Singleton<TTrulySecretsProducer>(std::move(ctx))->Get();
                case ESecretType::All:
                    return Singleton<TAllSecretsProducer>(std::move(ctx))->Get();
                case ESecretType::AllWMds:
                    return Singleton<TAllWMdsSecretsProducer>(std::move(ctx))->Get();
                default:
                    return Singleton<TGenericProducer>(std::move(ctx))->Get(neededSecrets);
            }
        }
    }

    THolder<TSearcher> TSnooper::Searcher(TSecretTypes neededSecrets) {
        return MakeHolder<TSearcher>(searchersStore(this->ctx, neededSecrets));
    }

    TSearcher* TSnooper::NewSearcher(TSecretTypes neededSecrets) {
        return new TSearcher(searchersStore(this->ctx, neededSecrets));
    }

    THolder<TProtoSearcher> TSnooper::ProtoSearcher(TSecretTypes neededSecrets) {
        return MakeHolder<TProtoSearcher>(searchersStore(this->ctx, neededSecrets));
    }
}
