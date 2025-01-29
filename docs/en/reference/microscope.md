# Microscope

Perforator storage can drop some profiles to save space. This feature is disabled by default but should be enabled for huge production installations. Microscope is a tool that forces Perforator storage to store all profiles satisfying the given selector. This is useful for more precise profiling for some period of time. Also, microscope requires user authorization to be enabled. Users can set up Microscope by running `perforator microscope` command.

Each microscope is saved in Postgres database. Each Perforator storage pod polls the database for new microscopes and saves profiles if they match the microscope selector. For now only concrete selectors are supported - by pod_id or node_id. This way storage can apply multiple microscopes at once by using node_id or pod_id maps. This approach allows constant complexity for microscope checks during `PushProfile` rpc calls.
