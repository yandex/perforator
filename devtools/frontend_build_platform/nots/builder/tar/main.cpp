#include <contrib/libs/libarchive/libarchive/archive.h>
#include <contrib/libs/libarchive/libarchive/archive_entry.h>

#include <util/generic/scope.h>
#include <util/generic/yexception.h>
#include <util/system/file.h>
#include <util/system/tempfile.h>
#include <util/thread/pool.h>

#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <iostream>
#include <future>
#include <map>
#include <memory>
#include <string>
#include <string_view>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

namespace {
    namespace fs = std::filesystem;

    void Check(archive* handle, int result) {
        if (result != ARCHIVE_OK) {
            const char* error = archive_error_string(handle);
            ythrow yexception() << (error ? error : "libarchive error");
        }
    }

    std::string ArchivePath(const char* name) {
        Y_ENSURE(name && *name && *name != '/', "Expected relative archive path");
        std::string result;
        std::string_view rest(name);
        while (!rest.empty()) {
            const auto slash = rest.find('/');
            const auto component = rest.substr(0, slash);
            Y_ENSURE(component != "..", "Parent traversal in archive path: " << name);
            if (!component.empty() && component != ".") {
                if (!result.empty()) {
                    result += '/';
                }
                result += component;
            }
            if (slash == rest.npos) {
                break;
            }
            rest.remove_prefix(slash + 1);
        }
        return result;
    }

    std::string Parent(const std::string& path) {
        const auto slash = path.rfind('/');
        return slash == path.npos ? std::string{} : path.substr(0, slash);
    }

    struct TMetadata {
        mode_t Mode = 0755;
        std::array<timespec, 2> Times = {{{0, UTIME_OMIT}, {0, UTIME_OMIT}}};

        TMetadata() = default;

        explicit TMetadata(archive_entry* entry)
            : Mode(archive_entry_perm(entry))
        {
            if (archive_entry_mtime_is_set(entry)) {
                Times[1] = {archive_entry_mtime(entry), archive_entry_mtime_nsec(entry)};
            }
        }
    };

    void CheckFile(int result, const char* operation, const std::string& path) {
        if (result == -1) {
            ythrow TSystemError() << operation << ": " << path.c_str();
        }
    }

    class TFileWorkers {
    public:
        template <class T>
        void Run(size_t count, T process) {
            if (count < 128) {
                process(0, count);
                return;
            }
            constexpr size_t workers = 4;
            if (!Started) {
                Pool.Start(workers);
                Started = true;
            }

            std::array<std::future<void>, workers> results;
            for (size_t i = 0; i < workers; ++i) {
                std::packaged_task<void()> task([process, i, count] {
                    process(i * count / workers, (i + 1) * count / workers);
                });
                results[i] = task.get_future();
                Pool.SafeAddFunc([task = std::move(task)]() mutable { task(); });
            }
            for (auto& result : results) {
                result.get();
            }
        }

    private:
        bool Started = false;
        TThreadPool Pool;
    };

    // Both tar writers use a single archive stream and commit entries in traversal order.
    class TArchiveWriter {
    public:
        TArchiveWriter(archive* output, fs::path destination, fs::path temporary)
            : Output(output)
            , Destination(std::move(destination))
            , Temporary(std::move(temporary))
        {
            Y_ENSURE(Entry, "Cannot allocate archive entry");
        }

        void AddRoot(const fs::path& source, const fs::path& name) {
            const auto path = fs::absolute(source).lexically_normal();
            if (fs::is_directory(fs::symlink_status(path))) {
                Walk(fs::canonical(path), name);
            } else {
                Y_ENSURE(!name.empty(), "A file needs a path in the archive");
                Queue(fs::canonical(path.parent_path()) / path.filename(), name);
            }
        }

        void Flush() {
            const size_t count = Pending.size();
            Workers.Run(count, [this](size_t begin, size_t end) {
                for (size_t i = begin; i < end; ++i) {
                    Prepare(Pending[i]);
                }
            });
            for (const auto& entry : Pending) {
                Write(entry);
            }
            Pending.clear();
        }

    private:
        struct TInput {
            fs::path Source;
            fs::path Name;
            struct stat Info = {};
            std::vector<char> Data;
        };

        void Walk(const fs::path& directory, const fs::path& name) {
            std::vector<fs::path> dirs;
            std::vector<fs::path> files;
            for (const auto& child : fs::directory_iterator(directory)) {
                (!child.is_symlink() && child.is_directory() ? dirs : files).push_back(child.path().filename());
            }
            std::sort(dirs.begin(), dirs.end());
            std::sort(files.begin(), files.end());
            for (const auto& child : dirs) {
                Queue(directory / child, name / child);
            }
            for (const auto& child : files) {
                Queue(directory / child, name / child);
            }
            for (const auto& child : dirs) {
                Walk(directory / child, name / child);
            }
        }

        void Queue(fs::path source, fs::path name) {
            if (source == Destination || source == Temporary || !Paths.insert(name.native()).second) {
                return;
            }
            Pending.push_back({std::move(source), std::move(name), {}, {}});
            if (Pending.size() == 1024) {
                Flush();
            }
        }

        static void Prepare(TInput& entry) {
            if (lstat(entry.Source.c_str(), &entry.Info) != 0) {
                ythrow TSystemError() << "Cannot stat " << entry.Source.c_str();
            }
            const auto& info = entry.Info;
            constexpr off_t maxPrefetchSize = 64 * 1024;
            if (S_ISREG(info.st_mode) && info.st_nlink == 1 && info.st_size > 0 && info.st_size <= maxPrefetchSize) {
                entry.Data.resize(info.st_size);
                TFile file(entry.Source, OpenExisting | RdOnly);
                file.Load(entry.Data.data(), entry.Data.size());
            }
        }

        void WriteData(const char* data, size_t size) {
            const auto written = archive_write_data(Output, data, size);
            if (written < 0) {
                Check(Output, static_cast<int>(written));
            }
            Y_ENSURE(static_cast<size_t>(written) == size, "Incomplete archive write");
        }

        void Write(const TInput& input) {
            const auto& info = input.Info;
            const auto type = info.st_mode & S_IFMT;
            if (type != S_IFREG && type != S_IFDIR && type != S_IFLNK) {
                return;
            }

            archive_entry_clear(Entry.get());
            archive_entry_set_pathname(Entry.get(), input.Name.c_str());
            archive_entry_set_mode(Entry.get(), info.st_mode);
            archive_entry_set_mtime(Entry.get(), 0, 0);
            archive_entry_set_size(Entry.get(), type == S_IFREG ? info.st_size : 0);
            bool hardlink = false;
            if (type != S_IFDIR && info.st_nlink > 1) {
                const auto [first, inserted] = Links.try_emplace({info.st_dev, info.st_ino}, input.Name.native());
                if (!inserted) {
                    archive_entry_set_hardlink(Entry.get(), first->second.c_str());
                    archive_entry_set_size(Entry.get(), 0);
                    hardlink = true;
                }
            }
            if (type == S_IFLNK && !hardlink) {
                archive_entry_set_symlink(Entry.get(), fs::read_symlink(input.Source).c_str());
            }

            Check(Output, archive_write_header(Output, Entry.get()));
            if (!input.Data.empty()) {
                WriteData(input.Data.data(), input.Data.size());
            } else if (type == S_IFREG && !hardlink && info.st_size > 0) {
                TFile file(input.Source, OpenExisting | RdOnly);
                for (auto left = info.st_size; left > 0;) {
                    const size_t size = std::min<off_t>(left, Buffer.size());
                    file.Load(Buffer.data(), size);
                    WriteData(Buffer.data(), size);
                    left -= size;
                }
            }
            Check(Output, archive_write_finish_entry(Output));
        }

        archive* Output;
        fs::path Destination;
        fs::path Temporary;
        std::unique_ptr<archive_entry, decltype(&archive_entry_free)> Entry{archive_entry_new(), archive_entry_free};
        std::map<std::pair<dev_t, ino_t>, std::string> Links;
        std::unordered_set<std::string> Paths;
        std::array<char, 64 * 1024> Buffer;
        std::vector<TInput> Pending;
        TFileWorkers Workers;
    };

    void Pack(const fs::path& destination, bool compressed) {
        // NUL-delimited (source, archive path) pairs avoid argv limits and preserve filesystem bytes.
        std::vector<std::pair<std::string, std::string>> roots;
        for (std::string source, name; std::getline(std::cin, source, '\0');) {
            Y_ENSURE(!source.empty() && !std::cin.eof() && std::getline(std::cin, name, '\0') && !std::cin.eof(),
                     "Incomplete archive input");
            roots.emplace_back(ArchivePath(name.c_str()), std::move(source));
        }
        Y_ENSURE(std::cin.eof() && !std::cin.bad(), "Cannot read archive input");
        std::sort(roots.begin(), roots.end());

        const auto output = fs::canonical(fs::absolute(destination).parent_path()) / destination.filename();
        auto temporary = output.string() + ".XXXXXX";
        const int fd = mkstemp(temporary.data());
        CheckFile(fd, "Cannot create archive", output.native());
        const TTempFile cleanup(temporary.c_str());
        TFile file(fd);
        const auto mask = umask(0);
        umask(mask);
        CheckFile(fchmod(fd, 0666 & ~mask), "Cannot set archive mode", output.native());

        const std::unique_ptr<archive, decltype(&archive_write_free)> writer(archive_write_new(), archive_write_free);
        Y_ENSURE(writer, "Cannot allocate archive writer");
        auto* handle = writer.get();
        Check(handle, archive_write_set_format_gnutar(handle));
        if (compressed) {
            Check(handle, archive_write_add_filter_zstd(handle));
            Check(handle, archive_write_set_filter_option(handle, "zstd", "compression-level", "1"));
        }
        Check(handle, archive_write_open_fd(handle, file.GetHandle()));
        Check(handle, archive_write_set_bytes_in_last_block(handle, 1));
        TArchiveWriter tree(handle, output, temporary);
        for (const auto& [name, source] : roots) {
            tree.AddRoot(source, name);
        }
        tree.Flush();
        Check(handle, archive_write_close(handle));
        file.Close();
        fs::rename(temporary, output);
    }

    std::string OverlayPath(const char* name, const std::string& prefix) {
        auto path = ArchivePath(name);
        if (prefix.empty()) {
            return path;
        }
        if (path == prefix) {
            return {};
        }
        Y_ENSURE(path.starts_with(prefix) && path[prefix.size()] == '/',
                 "Archive entry is outside output prefix: " << name);
        return path.substr(prefix.size() + 1);
    }

    // Duplicate paths and dependent replacements require the ordered disk writer.
    bool CanExtractOverlayInParallel(archive* reader, const std::string& prefix) {
        std::unordered_map<std::string, mode_t> paths;
        std::unordered_set<std::string> directories;
#ifdef _PC_CASE_SENSITIVE
        const bool caseInsensitive = pathconf(".", _PC_CASE_SENSITIVE) != 1;
        std::unordered_map<std::string, std::string> spellings;
        const auto checkSpelling = [&](const std::string& path) {
            auto folded = path;
            for (char& c : folded) {
                // Unicode aliases depend on the destination filesystem's normalization.
                if (static_cast<unsigned char>(c) >= 128) {
                    return false;
                }
                if (caseInsensitive && c >= 'A' && c <= 'Z') {
                    c += 'a' - 'A';
                }
            }
            if (!caseInsensitive) {
                return true;
            }
            return spellings.try_emplace(std::move(folded), path).first->second == path;
        };
#endif
        const auto checkDirectory = [&](std::string path) {
            while (!path.empty() && directories.insert(path).second) {
#ifdef _PC_CASE_SENSITIVE
                if (!checkSpelling(path)) {
                    return false;
                }
#endif
                const auto archived = paths.find(path);
                if (archived != paths.end() && archived->second != AE_IFDIR) {
                    return false;
                }
                const auto status = fs::symlink_status(path);
                if (fs::exists(status) && !fs::is_directory(status)) {
                    return false;
                }
                path = Parent(path);
            }
            return true;
        };

        while (true) {
            archive_entry* entry;
            const int result = archive_read_next_header(reader, &entry);
            if (result == ARCHIVE_EOF) {
                return true;
            }
            Check(reader, result);
            const auto path = OverlayPath(archive_entry_pathname(entry), prefix);
            if (path.empty()) {
                continue;
            }
#ifdef _PC_CASE_SENSITIVE
            if (!checkSpelling(path)) {
                return false;
            }
#endif
            const char* hardlink = archive_entry_hardlink(entry);
            const auto type = hardlink ? 0 : archive_entry_filetype(entry);
            if (hardlink) {
                const auto target = paths.find(OverlayPath(hardlink, prefix));
                if (target == paths.end() || target->second != AE_IFREG || archive_entry_size(entry) > 0) {
                    return false;
                }
            } else if ((type != AE_IFREG && type != AE_IFDIR && type != AE_IFLNK) ||
                       archive_entry_sparse_count(entry) || archive_entry_atime_is_set(entry) ||
                       (archive_entry_perm(entry) & 07000)) {
                return false;
            }
            if (!paths.emplace(path, type).second || (type != AE_IFDIR && directories.contains(path)) ||
                !checkDirectory(type == AE_IFDIR ? path : Parent(path))) {
                return false;
            }
        }
    }

    class TParallelExtractor {
    public:
        explicit TParallelExtractor(archive* reader, const std::string& prefix = {}, bool keepExisting = false,
                                    const struct stat* source = nullptr)
            : Reader(reader)
            , Prefix(prefix)
            , Source(source)
            , KeepExisting(keepExisting)
        {
            if (Source) {
                const auto mask = umask(0);
                umask(mask);
                DirectoryMode = 0777 & ~mask;
            }
        }

        void Extract() {
            while (true) {
                archive_entry* entry;
                const int result = archive_read_next_header(Reader, &entry);
                if (result == ARCHIVE_EOF) {
                    break;
                }
                Check(Reader, result);
                Add(entry);
            }
            Flush();
            // The overlay preflight guarantees that link targets do not depend on other links.
            if (Source) {
                Workers.Run(Links.size(), [this](size_t begin, size_t end) {
                    for (size_t i = begin; i < end; ++i) {
                        WriteLink(Links[i]);
                    }
                });
            } else {
                for (const auto& link : Links) {
                    WriteLink(link);
                }
            }
            for (auto it = DirectoryOrder.rbegin(); it != DirectoryOrder.rend(); ++it) {
                const auto& directory = Directories.at(*it);
                if (directory.SetMode) {
                    CheckFile(chmod(it->c_str(), directory.Metadata.Mode), "Cannot set directory mode", *it);
                }
                if (directory.Metadata.Times[1].tv_nsec != UTIME_OMIT) {
                    CheckFile(utimensat(AT_FDCWD, it->c_str(), directory.Metadata.Times.data(), 0),
                              "Cannot set directory time", *it);
                }
            }
        }

    private:
        struct TPendingFile {
            std::string Path;
            TMetadata Metadata;
            std::vector<char> Data;
        };

        struct TPendingLink {
            std::string Path;
            std::string Target;
            TMetadata Metadata;
            bool Hard;
        };

        struct TDirectory {
            TMetadata Metadata;
            bool SetMode = true;
        };

        void EnsureDirectory(const std::string& path) {
            if (path.empty() || !Directories.try_emplace(path).second) {
                return;
            }
            EnsureDirectory(Parent(path));
            auto& directory = Directories.at(path);
            if (Source && fs::is_directory(fs::symlink_status(path))) {
                directory.SetMode = false;
            } else {
                CheckFile(mkdir(path.c_str(), 0700), "Cannot create directory", path);
                directory.Metadata.Mode = DirectoryMode;
            }
            DirectoryOrder.push_back(path);
        }

        void RemoveExisting(const std::string& path) const {
            if (!Source || KeepExisting) {
                return;
            }
            struct stat existing;
            if (lstat(path.c_str(), &existing) == -1) {
                if (errno == ENOENT) {
                    return;
                }
                CheckFile(-1, "Cannot inspect destination", path);
            }
            Y_ENSURE(existing.st_dev != Source->st_dev || existing.st_ino != Source->st_ino,
                     "Refusing to overwrite archive: " << path.c_str());
            const int result = S_ISDIR(existing.st_mode) ? rmdir(path.c_str()) : unlink(path.c_str());
            CheckFile(result, "Cannot replace destination", path);
        }

        void WriteLink(const TPendingLink& link) const {
            RemoveExisting(link.Path);
            if (link.Hard) {
                Y_ENSURE(Paths.contains(link.Target) && !Directories.contains(link.Target),
                         "Hardlink target must be an archive file: " << link.Target.c_str());
                CheckFile(linkat(AT_FDCWD, link.Target.c_str(), AT_FDCWD, link.Path.c_str(), 0),
                          "Cannot create hardlink", link.Path);
                return;
            }
            CheckFile(symlink(link.Target.c_str(), link.Path.c_str()), "Cannot create symlink", link.Path);
#ifdef __APPLE__
            if (link.Metadata.Mode != 0777) {
                CheckFile(lchmod(link.Path.c_str(), link.Metadata.Mode), "Cannot set symlink mode", link.Path);
            }
#endif
            CheckFile(utimensat(AT_FDCWD, link.Path.c_str(), link.Metadata.Times.data(), AT_SYMLINK_NOFOLLOW),
                      "Cannot set symlink time", link.Path);
        }

        TFile Open(const std::string& path) const {
            const auto create = [&] {
                int fd;
                do {
                    fd = open(path.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600);
                } while (fd == -1 && errno == EINTR);
                return fd;
            };
            int fd = create();
            if (fd == -1 && Source && !KeepExisting && (errno == EEXIST || errno == EISDIR)) {
                RemoveExisting(path);
                fd = create();
            }
            CheckFile(fd, "Cannot create file", path);
            return TFile(fd);
        }

        static void Finish(TFile& file, const std::string& path, const TMetadata& metadata) {
            CheckFile(fchmod(file.GetHandle(), metadata.Mode), "Cannot set file mode", path);
            CheckFile(futimens(file.GetHandle(), metadata.Times.data()), "Cannot set file time", path);
            file.Close();
        }

        void Read(char* data, size_t size) {
            while (size) {
                const auto count = archive_read_data(Reader, data, size);
                if (count < 0) {
                    Check(Reader, static_cast<int>(count));
                }
                Y_ENSURE(count > 0, "Truncated archive entry");
                data += count;
                size -= count;
            }
        }

        void Add(archive_entry* entry) {
            auto path = OverlayPath(archive_entry_pathname(entry), Prefix);
            if (Source && path.empty()) {
                return;
            }
            const auto type = archive_entry_filetype(entry);
            TMetadata metadata(entry);
            Y_ENSURE(Paths.insert(path).second, "Duplicate archive path: " << path.c_str());
            if (KeepExisting && fs::exists(fs::symlink_status(path))) {
                return;
            }
            if (type == AE_IFDIR && !archive_entry_hardlink(entry)) {
                const bool implicit = Directories.contains(path);
                EnsureDirectory(path);
                if (path.empty()) {
                    path = ".";
                    DirectoryOrder.insert(DirectoryOrder.begin(), path);
                }
                auto& directory = Directories[path];
                if (Source && implicit) {
                    // The ordered writer preserves the mode of a directory created for an earlier child.
                    metadata.Mode = directory.Metadata.Mode;
                }
                directory.Metadata = metadata;
                return;
            }

            Y_ENSURE(!path.empty(), "Expected non-empty archive path");
            EnsureDirectory(Parent(path));
            if (const char* target = archive_entry_hardlink(entry)) {
                auto link = OverlayPath(target, Prefix);
                Y_ENSURE(!link.empty(), "Expected non-empty hardlink target");
                Links.push_back({std::move(path), std::move(link), metadata, true});
                return;
            }
            if (type == AE_IFLNK) {
                const char* target = archive_entry_symlink(entry);
                Y_ENSURE(target, "Missing symlink target");
                Links.push_back({std::move(path), target, metadata, false});
                return;
            }

            Y_ENSURE(type == AE_IFREG, "Unsupported archive entry type");
            auto size = archive_entry_size(entry);
            Y_ENSURE(size >= 0, "Invalid file size");
            constexpr la_int64_t maxFileBuffer = 1024 * 1024;
            if (size > maxFileBuffer) {
                auto file = Open(path);
                std::array<char, 64 * 1024> buffer;
                while (size) {
                    const size_t count = std::min<la_int64_t>(buffer.size(), size);
                    Read(buffer.data(), count);
                    file.Write(buffer.data(), count);
                    size -= count;
                }
                Finish(file, path, metadata);
                return;
            }

            Pending.push_back({std::move(path), metadata, std::vector<char>(size)});
            Read(Pending.back().Data.data(), size);
            PendingBytes += size;
            if (Pending.size() == 1024 || PendingBytes >= 32 * 1024 * 1024) {
                Flush();
            }
        }

        void Flush() {
            Workers.Run(Pending.size(), [this](size_t begin, size_t end) {
                for (size_t i = begin; i < end; ++i) {
                    const auto& entry = Pending[i];
                    auto file = Open(entry.Path);
                    if (!entry.Data.empty()) {
                        file.Write(entry.Data.data(), entry.Data.size());
                    }
                    Finish(file, entry.Path, entry.Metadata);
                }
            });
            Pending.clear();
            PendingBytes = 0;
        }

        archive* Reader;
        std::string Prefix;
        const struct stat* Source;
        bool KeepExisting;
        mode_t DirectoryMode = 0755;
        std::unordered_set<std::string> Paths;
        std::unordered_map<std::string, TDirectory> Directories;
        std::vector<std::string> DirectoryOrder;
        std::vector<TPendingLink> Links;
        std::vector<TPendingFile> Pending;
        size_t PendingBytes = 0;
        // Join queued jobs before destroying their input on failure.
        TFileWorkers Workers;
    };

    void ExtractOverlay(archive* reader, const std::string& prefix, bool keepExisting) {
        const std::unique_ptr<archive, decltype(&archive_write_free)> disk(archive_write_disk_new(), archive_write_free);
        Y_ENSURE(disk, "Cannot allocate disk writer");
        constexpr int flags = ARCHIVE_EXTRACT_TIME | ARCHIVE_EXTRACT_SECURE_SYMLINKS |
                              ARCHIVE_EXTRACT_SECURE_NODOTDOT | ARCHIVE_EXTRACT_SECURE_NOABSOLUTEPATHS;
        while (true) {
            archive_entry* entry;
            const int result = archive_read_next_header(reader, &entry);
            if (result == ARCHIVE_EOF) {
                break;
            }
            Check(reader, result);
            const auto path = OverlayPath(archive_entry_pathname(entry), prefix);
            if (path.empty()) {
                continue;
            }

            const auto type = archive_entry_filetype(entry);
            const char* target = archive_entry_hardlink(entry);
            Y_ENSURE(target || type == AE_IFDIR || type == AE_IFREG || type == AE_IFLNK, "Unsupported archive entry type");
            if (target) {
                const auto hardlink = OverlayPath(target, prefix);
                Y_ENSURE(!hardlink.empty(), "Expected non-empty hardlink target");
                archive_entry_set_hardlink(entry, hardlink.c_str());
            }
            const auto status = fs::symlink_status(path);
            if (keepExisting && fs::exists(status)) {
                continue;
            }
            archive_entry_set_pathname(entry, path.c_str());
            // Merging an archive directory must not change an existing directory's permissions.
            const bool existingDirectory = type == AE_IFDIR && fs::is_directory(status);
            const int options = flags | (existingDirectory ? 0 : ARCHIVE_EXTRACT_PERM);
            Check(disk.get(), archive_write_disk_set_options(disk.get(), options));
            Check(reader, archive_read_extract2(reader, entry, disk.get()));
        }
        Check(disk.get(), archive_write_close(disk.get()));
    }

    auto OpenTar(TFile& file, bool layer) {
        std::unique_ptr<archive, decltype(&archive_read_free)> reader(archive_read_new(), archive_read_free);
        Y_ENSURE(reader, "Cannot allocate archive reader");
        auto* handle = reader.get();
        Check(handle, archive_read_support_format_tar(handle));
        // Keep AppleDouble files as ordinary entries on macOS as well.
        Check(handle, archive_read_set_format_option(handle, "tar", "mac-ext", nullptr));
        if (layer) {
            Check(handle, archive_read_support_filter_zstd(handle));
        }
        Check(handle, archive_read_open_fd(handle, file.GetHandle(), 10240));
        return reader;
    }

    void Unpack(const fs::path& source, const fs::path& destination, bool layer, const char* prefix = "",
                bool keepExisting = false) {
        TFile file(source, OpenExisting | RdOnly);
        auto reader = OpenTar(file, layer);
        const auto output = fs::absolute(destination);
        const auto cwd = fs::current_path();
        if (layer) {
            Y_ENSURE(fs::create_directory(output), "Destination must be a new directory");
        }
        Y_DEFER {
            std::error_code error;
            fs::current_path(cwd, error);
        };
        fs::current_path(output);
        if (layer) {
            TParallelExtractor(reader.get()).Extract();
        } else {
            const auto normalized = *prefix ? ArchivePath(prefix) : std::string{};
            struct stat info;
            CheckFile(fstat(file.GetHandle(), &info), "Cannot stat archive", source.native());
            const bool regular = S_ISREG(info.st_mode);
            const bool parallel = regular && CanExtractOverlayInParallel(reader.get(), normalized);
            if (regular) {
                Check(reader.get(), archive_read_close(reader.get()));
                file.Seek(0, sSet);
                reader = OpenTar(file, false);
            }
            if (parallel) {
                TParallelExtractor(reader.get(), normalized, keepExisting, &info).Extract();
            } else {
                ExtractOverlay(reader.get(), normalized, keepExisting);
            }
        }
        Check(reader.get(), archive_read_close(reader.get()));
    }
} // namespace

int main(int argc, char** argv) {
    const std::string_view operation = argc > 1 ? argv[1] : "";
    try {
        if (argc == 3 && (operation == "pack-tar" || operation == "pack-layer")) {
            Pack(argv[2], operation == "pack-layer");
        } else if (argc == 4 && operation == "unpack-layer") {
            Unpack(argv[2], argv[3], true);
        } else if ((argc == 5 || (argc == 6 && std::string_view(argv[5]) == "--keep-existing")) &&
                   operation == "unpack-tar") {
            Unpack(argv[2], argv[3], false, argv[4], argc == 6);
        } else {
            std::fprintf(stderr, "Usage: nots-tar (pack-tar|pack-layer) OUTPUT < NUL_DELIMITED_INPUTS\n"
                                 "       nots-tar unpack-layer ARCHIVE NEW_DIRECTORY\n"
                                 "       nots-tar unpack-tar ARCHIVE DIRECTORY PREFIX [--keep-existing]\n");
            return EXIT_FAILURE;
        }
    } catch (const std::exception& error) {
        std::fprintf(stderr, "%s\n", error.what());
        return EXIT_FAILURE;
    }
    return EXIT_SUCCESS;
}
