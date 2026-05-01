#include "snooper.h"

#include <util/generic/string.h>
#include <util/generic/strbuf.h>
#include <util/generic/vector.h>
#include <util/generic/singleton.h>

#include <security/ant-secret/snooper/cpp/snooper.h>
#include <security/ant-secret/snooper/cpp/searcher.h>
#include <security/ant-secret/snooper/cpp/secret.h>


namespace NPySnooper {
    namespace {
        constexpr int kSearchPresetTrulySecrets = 1;
        constexpr int kSearchPresetAll          = 2;
        constexpr int kSearchPresetExtra        = 3;
    }

    class TSearcher : public Py::PythonClass<TSearcher>
    {
    public:
        TSearcher(Py::PythonClassInstance *self, Py::Tuple& args, Py::Dict& kwargs)
            : Py::PythonClass<TSearcher>::PythonClass(self, args, kwargs)
        {
            Y_UNUSED(self);

            int pyPreset = 0;
            if (XPython::HasArgument(args, kwargs, "preset")) {
                pyPreset = static_cast<int>(Py::Int(XPython::ExtractArgument(args, kwargs, "preset")));
            }

            NSecret::ESecretType preset;
            switch (pyPreset) {
            case kSearchPresetTrulySecrets:
                preset = NSnooper::ESecretType::TrulySecrets;
                break;

            case kSearchPresetAll:
                preset = NSnooper::ESecretType::All;
                break;

            case kSearchPresetExtra:
                preset = NSnooper::ESecretType::AllWMds;
                break;

            default:
                preset = NSnooper::ESecretType::All;
                break;
            }

            this->searcher_ = NSnooper::TSnooper().Searcher(preset);
        }

        Py::Object Search(const Py::Tuple& args_, const Py::Dict& kwargs_)
        {
            auto args = args_;
            auto kwargs = kwargs_;

            bool validOnly = false;
            if (XPython::HasArgument(args, kwargs, "valid_only")) {
                auto arg = XPython::ExtractArgument(args, kwargs, "valid_only");
                validOnly = Py::Boolean(arg);
            }

            auto dataObj = XPython::ExtractArgument(args, kwargs, "data");
            TString data = XPython::ConvertStringObjectToString(dataObj);
            NSnooper::TSecretList secrets;
            try {
                secrets = this->searcher_->Search(data, validOnly);
            } catch (const std::exception& ex) {
                throw Py::RuntimeError(TString("search secrets: ") + ex.what());
            } catch (...) {
                throw Py::RuntimeError("unable to search secrets");
            }

            Py::List pySecrets(secrets.size());
            for (size_t i = 0; i < secrets.size(); ++i) {
                Py::Dict pySecret;
                pySecret.setItem("type", Py::Long(static_cast<int>(secrets[i].Type)));
                pySecret.setItem("secret", XPython::ConvertToPythonString(secrets[i].Secret));
                pySecret.setItem("secret_from", Py::Long(secrets[i].SecretPos.From));
                pySecret.setItem("secret_len", Py::Long(secrets[i].SecretPos.Len));
                pySecret.setItem("mask_from", Py::Long(secrets[i].MaskPos.From));
                pySecret.setItem("mask_len", Py::Long(secrets[i].MaskPos.Len));

                pySecrets[i] = pySecret;
            }

            return pySecrets;
        }
        PYCXX_KEYWORDS_METHOD_DECL(TSearcher, Search)

        static void InitType()
        {
            behaviors().name("ant_secret_snooper_bindings.Searcher");
            behaviors().doc("Search some secrets");

            PYCXX_ADD_KEYWORDS_METHOD(search, Search, "Search secrets");
            behaviors().readyType();
        }

    private:
        THolder<NSnooper::TSearcher> searcher_;
    };

    class TSnooperModule
        : public Py::ExtensionModule<TSnooperModule>
    {
    public:
        TSnooperModule()
        : Py::ExtensionModule<TSnooperModule>("ant_secret_snooper_bindings")
        {
    #if PY_VERSION_HEX < 0x03090000
            PyEval_InitThreads();
    #endif

            TSearcher::InitType();

            initialize("Python bindings for Snooper");

            Py::Dict moduleDict(moduleDictionary());
            Py::Object searcherClass(TSearcher::type());
            moduleDict.setItem("Searcher", searcherClass);
        }

        virtual ~TSnooperModule()
        { }
    };

} // namespace NPySnooper


PyObject* NPySnooper::DoInit()
{
  static NPySnooper::TSnooperModule* snooper_mod = new NPySnooper::TSnooperModule;
  return snooper_mod->module().ptr();
}
