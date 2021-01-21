--liquibase formatted sql

-- The versioning is maintained using liquibase. New change set must be appended at the end of this file with a new change set.
-- Only the incremental changes are applied by keeping an applied version in a book-keeping table in each database.
-- We will not allow rollback for now.

-- Create the TenantModel
--changeset sherlock:1
create table if not exists tenant_model (
    id varchar(36) primary key,
    version bigint not null,
    name varchar(200) not null,
    token varchar(4096) not null,
    description varchar(200),
    created_at timestamp not null,
    updated_at timestamp not null);

-- Create the CategoryModel
create table if not exists category_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    purpose varchar(200),
    created_at timestamp not null,
    updated_at timestamp not null);

alter table category_model drop constraint if exists category_unique_name;
alter table category_model add constraint category_unique_name unique (name, tenant_id);

-- Create the CategoryModel to hold values for the categories
create table if not exists category_value_model (
    id bigserial primary key,
    category_id varchar(36) not null references category_model(id) ON DELETE CASCADE,
    value varchar(200) not null);

alter table category_value_model drop constraint if exists category_value_unique_value;
alter table category_value_model add constraint category_value_unique_value unique (value, category_id);

-- Create the CloudCredsModel to hold AWS or GCP cloud records
create table if not exists cloud_creds_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    type varchar(20) not null,
    aws_credential varchar(200),
    gcp_credential varchar(4096),
    created_at timestamp not null,
    updated_at timestamp not null);

alter table cloud_creds_model drop constraint if exists cloud_creds_model_unique_name;
alter table cloud_creds_model add constraint cloud_creds_model_unique_name unique (name, tenant_id);

-- Create DockerProfileModel to hold Container Registry Data
create table if not exists docker_profile_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    type varchar(36),
    credentials varchar(4096),
    server varchar(512),
    user_name varchar(36),
    email varchar(36),
    pwd varchar(36),
    cloud_creds_id varchar(36),
    created_at timestamp,
    updated_at timestamp);


-- Create the EdgeModel to hold edges
create table if not exists edge_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    serial_number varchar(200) not null,
    ip_address varchar(20),
    gateway varchar(20),
    subnet varchar(20),
    edge_devices integer,
    storage_capacity bigint,
    storage_usage bigint,
    connected boolean,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table edge_model drop constraint if exists edge_model_unique_name;
alter table edge_model add constraint edge_model_unique_name unique (name, tenant_id);

-- Create the edge_cert_model
create table if not exists edge_cert_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE,
    certificate varchar(4096),
    private_key varchar(4096),
    locked boolean,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table edge_cert_model drop constraint if exists edge_cert_model_unique_edge;
alter table edge_cert_model add constraint edge_cert_model_unique_edge unique (edge_id);

-- Create the DataSourceModel to hold data sources
create table if not exists data_source_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id),
    name varchar(200) not null,
    type varchar(20),
    sensor_model varchar(200),
    connection varchar(20),
    protocol varchar(20),
    auth_type varchar(20),
    created_at timestamp not null,
    updated_at timestamp not null);

alter table data_source_model drop constraint if exists data_source_model_unique_name;
alter table data_source_model add constraint data_source_model_unique_name unique (name, tenant_id);

-- Create the DataSourceFieldsModel to hold fields for the data source
create table if not exists data_source_field_model (
    id serial primary key,
    data_source_id varchar(36) not null references data_source_model(id) ON DELETE CASCADE,
    name varchar(200) not null,
    mqtt_topic varchar(4096),
    field_type varchar(20));

alter table data_source_field_model drop constraint if exists data_source_field_model_unique_name;
alter table data_source_field_model add constraint data_source_field_model_unique_name unique (name, data_source_id);

-- Create the DataSourceFieldSelectorModel to select fields on the data source
-- fieldId is nullable to indicate all values 
create table if not exists data_source_field_selector_model (
    id serial primary key,
    data_source_id varchar(36) not null references data_source_model(id) ON DELETE CASCADE,
    field_id integer references data_source_field_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id));

alter table data_source_field_selector_model drop constraint if exists data_source_field_selector_model_unique_fields;
alter table data_source_field_selector_model add constraint data_source_field_selector_model_unique_fields unique (
    category_value_id,
    data_source_id,
    field_id);


-- Create the DataStreamModel to hold data streams
create table if not exists data_stream_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    description varchar(200),
    name varchar(200) not null,
    data_type varchar(20),
    origin varchar(20),
    origin_id varchar(36),
    destination varchar(20),
    cloud_type varchar(20),
    cloud_creds_id varchar(36) references cloud_creds_model(id),
    aws_cloud_region varchar(100),
    gcp_cloud_region varchar(100),
    edge_stream_type varchar(20),
    aws_stream_type varchar(20),
    gcp_stream_type varchar(20),
    size integer,
    enable_sampling boolean,
    sampling_interval integer,
    transformation_args_list varchar(4096),
    data_retention varchar(200),
    created_at timestamp not null,
    updated_at timestamp not null);

alter table data_stream_model drop constraint if exists data_stream_model_unique_name;
alter table data_stream_model add constraint data_stream_model_unique_name unique (tenant_id, name);

create table if not exists data_stream_origin_model (
    id serial primary key,
    data_stream_id varchar(36) not null references data_stream_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id));

alter table data_stream_origin_model drop constraint if exists data_stream_origin_model_unique_category;
alter table data_stream_origin_model add constraint data_stream_origin_model_unique_category unique (
    data_stream_id,
    category_value_id);

-- Create the ApplicationModel to hold applications
create table if not exists application_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    description varchar(200),
    name varchar(200) not null,
    yaml_data varchar(30720) not null,
    created_at timestamp,
    updated_at timestamp);

alter table application_model drop constraint if exists application_model_unique_name;
alter table application_model add constraint application_model_unique_name unique (name, tenant_id);

-- Create the SensorModel to hold sensors
create table if not exists sensor_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id),
    topic_name varchar(4096),
    created_at timestamp not null,
    updated_at timestamp not null);


-- Create the ScriptModel to hold scripts
create table if not exists script_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    description varchar(200),
    name varchar(200) not null,
    type varchar(20),
    language varchar(20) not null,
    environment varchar(20) not null,
    code varchar(30720) not null,
    params varchar(4096),
    created_at timestamp,
    updated_at timestamp);

alter table script_model drop constraint if exists script_model_unique_name;
alter table script_model add constraint script_model_unique_name unique (tenant_id, name);


-- Create the LogModel to hold log records
create table if not exists log_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    batch_id varchar(36) not null,
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE,
    location varchar(4096) not null unique,
    status varchar(20) not null,
    error_message varchar(200),
    created_at timestamp not null,
    updated_at timestamp not null);

-- Create the UserModel to hold user records
create table if not exists user_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    email varchar(200) unique not null,
    password varchar(200),
    created_at timestamp not null,
    updated_at timestamp not null);

-- Because of previous types.Bool, null was inserted for false.
--changeset sherlock:2
update edge_model set connected = false where connected is null;

-- Create the ScriptRuntimeModel to hold script runtime records
create table if not exists script_runtime_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    language varchar(100) not null,
    docker_repo_uri varchar(4096),
    docker_profile_id varchar(36),
    dockerfile varchar(4096),
    builtin boolean,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table script_runtime_model drop constraint if exists script_runtime_model_unique_name;
alter table script_runtime_model add constraint script_runtime_model_unique_name unique (tenant_id, name);


--changeset sherlock:3
-- Create the ApplicationStatusModel to hold applications status
create table if not exists application_status_model (
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id),
    application_id varchar(36) not null references application_model(id),
    version bigint not null,
    app_status varchar(30720) not null,
    created_at timestamp,
    updated_at timestamp,
    PRIMARY KEY(tenant_id, edge_id, application_id));

--changeset sherlock:4
-- ON DELETE CASCADE for ApplicationStatusModel foreign keys
alter table application_status_model drop constraint if exists application_status_model_tenant_id_fkey;
alter table application_status_model drop constraint if exists application_status_model_edge_id_fkey;
alter table application_status_model drop constraint if exists application_status_model_application_id_fkey;
alter table application_status_model add constraint application_status_model_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenant_model(id) ON DELETE CASCADE;
alter table application_status_model add constraint application_status_model_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
alter table application_status_model add constraint application_status_model_application_id_fkey FOREIGN KEY (application_id) REFERENCES application_model(id) ON DELETE CASCADE;

--changeset sherlock:5
-- ON DELETE CASCADE for DockerProfileModel foreign keys
alter table docker_profile_model drop constraint if exists docker_profile_model_cloud_creds_id_fkey;
alter table docker_profile_model add constraint docker_profile_model_cloud_creds_id_fkey FOREIGN KEY (cloud_creds_id) REFERENCES cloud_creds_model(id)  ON DELETE CASCADE;

--changeset sherlock:6
-- Add columns to ScriptModel
alter table script_model ADD COLUMN runtime_id varchar(36) references script_runtime_model(id),
    ADD COLUMN runtime_tag varchar(128);

--changeset sherlock:7
-- Increase ScriptRuntimeModel id length
ALTER TABLE script_runtime_model ALTER COLUMN id TYPE VARCHAR(64);


--changeset sherlock:8
-- Create EdgeInfoModel to hold edge info
create table if not exists edge_info_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE,
    num_cpu varchar(20), 
    total_memory_kb varchar(20), 
    total_storage_kb varchar(20), 
    gpu_info varchar(20), 
    cpu_usage varchar(20), 
    memory_free_kb varchar(20), 
    storage_free_kb varchar(20), 
    gpu_usage varchar(20), 
    created_at timestamp not null, 
    updated_at timestamp not null);

alter table edge_info_model drop constraint if exists edge_info_model_unique_edge;
alter table edge_info_model add constraint edge_info_model_unique_edge unique (edge_id);

--changeset sherlock:9
-- Increase CategoryModel id length
ALTER TABLE category_model ALTER COLUMN id TYPE VARCHAR(64);
ALTER TABLE category_value_model ALTER COLUMN category_id TYPE VARCHAR(64);

--changeset sherlock:10
-- Add tag column to hold properties of the log entry
alter table log_model add column tags varchar(1024);

--changeset sherlock:11
-- Create ProjectModel to hold project
create table if not exists project_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id) ON DELETE CASCADE,
    name varchar(200) not null,
    description varchar(200) not null,
    edge_selector_type varchar(20),
    edge_ids varchar(4096),
    edge_selectors varchar(4096),
    created_at timestamp not null,
    updated_at timestamp not null);

-- Create the ProjectUserModel to hold projects <-> users relation
create table if not exists project_user_model (
    id bigserial primary key,
    project_id varchar(36) not null references project_model(id) ON DELETE CASCADE,
    user_id varchar(36) not null references user_model(id) ON DELETE CASCADE,
    user_role varchar(36) not null);

-- Create the ProjectCloudCredsModel to hold projects <-> cloud creds relation
create table if not exists project_cloud_creds_model (
    id bigserial primary key,
    project_id varchar(36) not null references project_model(id) ON DELETE CASCADE,
    cloud_creds_id varchar(36) not null references cloud_creds_model(id) ON DELETE CASCADE);

-- Create the ProjectDockerProfileModel to hold projects <-> docker profiles relation
create table if not exists project_docker_profile_model (
    id bigserial primary key,
    project_id varchar(36) not null references project_model(id) ON DELETE CASCADE,
    docker_profile_id varchar(36) not null references docker_profile_model(id) ON DELETE CASCADE);

alter table user_model ADD COLUMN role varchar(36);
alter table script_model ADD COLUMN builtin boolean,
    ADD COLUMN project_id varchar(36) references project_model(id) ON DELETE CASCADE;
alter table script_runtime_model ADD COLUMN project_id varchar(36) references project_model(id) ON DELETE CASCADE;
alter table application_model ADD COLUMN project_id varchar(36) references project_model(id) ON DELETE CASCADE;
alter table data_stream_model ADD COLUMN project_id varchar(36) references project_model(id) ON DELETE CASCADE;

--changeset sherlock:12
-- Change ProjectModel id length to 64
ALTER TABLE project_model ALTER COLUMN id TYPE VARCHAR (64);
ALTER TABLE project_user_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE project_cloud_creds_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE project_docker_profile_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE script_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE script_runtime_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE application_model ALTER COLUMN project_id TYPE VARCHAR (64);
ALTER TABLE data_stream_model ALTER COLUMN project_id TYPE VARCHAR (64);

--changeset sherlock:13
-- Create DomainModel to hold domain to tenant mapping
create table if not exists domain_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200) not null,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table domain_model drop constraint if exists domain_model_unique_name;
alter table domain_model add constraint domain_model_unique_name unique (name);

--changeset sherlock:14
-- DockerProfile and CloudCreds password encryption changes
-- Change DockerProfileModel pwd length to 256
alter table docker_profile_model ALTER COLUMN pwd TYPE VARCHAR (256);
-- Add iflag_encrypted column to DockerProfile and CloudCreds
alter table docker_profile_model ADD COLUMN iflag_encrypted boolean;
alter table cloud_creds_model ADD COLUMN iflag_encrypted boolean;
alter table cloud_creds_model ALTER COLUMN aws_credential TYPE VARCHAR (500);


--changeset sherlock:15
-- DockerProfile unique name
alter table docker_profile_model drop constraint if exists docker_profile_model_unique_name;
alter table docker_profile_model add constraint docker_profile_model_unique_name unique (name, tenant_id);

--changeset sherlock:16
-- Increase docker profile password length to 8192
-- since for GCP we are putting entire GCP credential into it
alter table docker_profile_model ALTER COLUMN pwd TYPE VARCHAR (8192);

--changeset sherlock:17
-- Add the edge_version to edge_info model
alter table edge_info_model ADD COLUMN edge_version varchar(20);

--changeset sherlock:18
-- Add edge_ids to application_model
alter table application_model ADD COLUMN edge_ids varchar(4096);

--changeset sherlock:19
-- Increase docker profile email length to 200
-- Increase docker profile user_name length to 200
alter table docker_profile_model ALTER COLUMN email TYPE VARCHAR (200);
alter table docker_profile_model ALTER COLUMN user_name TYPE VARCHAR (200);


--changeset sherlock:20
-- Add the edge_build_num to edge_info model
alter table edge_info_model ADD COLUMN edge_build_num varchar(20);


--changeset sherlock:21
-- Create the project_edge_model to hold projects <-> edges relation
create table if not exists project_edge_model (
    id bigserial primary key,
    project_id varchar(36) not null references project_model(id) ON DELETE CASCADE,
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE);
-- Create the application_edge_model to hold applications <-> edges relation
create table if not exists application_edge_model (
    id bigserial primary key,
    application_id varchar(36) not null references application_model(id) ON DELETE CASCADE,
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE);
-- Note we do not migrate project and application table data since only test
-- currently have these
-- drop old columns
alter table project_model drop column edge_ids;
alter table application_model drop column edge_ids;

--changeset sherlock:22
-- Create the tenant_rootca_model to hold per-tenant root CA
create table if not exists tenant_rootca_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id) ON DELETE CASCADE,
    certificate varchar(4096) not null,
    private_key varchar(4096) not null,
    aws_data_key varchar(4096) not null,
    created_at timestamp not null,
    updated_at timestamp not null);

-- Add columns to edge_cert_model
alter table edge_cert_model ADD COLUMN client_certificate varchar(4096),
    ADD COLUMN client_private_key varchar(4096);

alter table tenant_rootca_model drop constraint if exists tenant_rootca_unique_tenant_id;
alter table tenant_rootca_model add constraint tenant_rootca_unique_tenant_id unique (tenant_id);

--changeset sherlock:23
-- Increase ScriptModel environment length
ALTER TABLE script_model ALTER COLUMN environment TYPE VARCHAR(4096);

--changeset sherlock:24
-- Add columns to edge_cert_model
alter table edge_cert_model ADD COLUMN edge_certificate varchar(4096),
    ADD COLUMN edge_private_key varchar(4096);

--changeset sherlock:25
alter table project_edge_model drop constraint if exists project_edge_unique_pair;
alter table project_edge_model add constraint project_edge_unique_pair unique (project_id, edge_id);

alter table project_user_model drop constraint if exists project_user_unique_pair;
alter table project_user_model add constraint project_user_unique_pair unique (project_id, user_id);

alter table project_cloud_creds_model drop constraint if exists project_cloud_creds_unique_pair;
alter table project_cloud_creds_model add constraint project_cloud_creds_unique_pair unique (project_id, cloud_creds_id);

alter table project_docker_profile_model drop constraint if exists project_docker_profile_unique_pair;
alter table project_docker_profile_model add constraint project_docker_profile_unique_pair unique (project_id, docker_profile_id);

alter table application_edge_model drop constraint if exists application_edge_unique_pair;
alter table application_edge_model add constraint application_edge_unique_pair unique (application_id, edge_id);

--changeset sherlock:26
alter table project_user_model drop constraint if exists project_user_unique_pair;
alter table project_user_model add constraint project_user_unique_pair unique (project_id, user_id, user_role);

--changeset sherlock:27
-- Edge category labels, category-based edge selection in projects and applications
create table if not exists edge_label_model (
    id serial primary key,
    edge_id varchar(36) not null references edge_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id) ON DELETE CASCADE);
create table if not exists project_edge_selector_model (
    id serial primary key,
    project_id varchar(36) not null references project_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id) ON DELETE CASCADE);
create table if not exists application_edge_selector_model (
    id serial primary key,
    application_id varchar(36) not null references application_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id) ON DELETE CASCADE);

alter table edge_label_model drop constraint if exists edge_label_unique_value;
alter table edge_label_model add constraint edge_label_unique_value unique (edge_id, category_value_id);

alter table project_edge_selector_model drop constraint if exists project_edge_selector_unique_value;
alter table project_edge_selector_model add constraint project_edge_selector_unique_value unique (project_id, category_value_id);

alter table application_edge_selector_model drop constraint if exists application_edge_selector_unique_value;
alter table application_edge_selector_model add constraint application_edge_selector_unique_value unique (application_id, category_value_id);

-- changeset sherlock:28
alter table edge_model drop constraint if exists edge_model_unique_serial;
alter table edge_model add constraint edge_model_unique_serial unique (serial_number);

alter table log_model drop constraint if exists log_model_unique_tenant_edge_batch;
alter table log_model add constraint log_model_unique_tenant_edge_batch unique (tenant_id, edge_id, batch_id);

create index category_tenant on category_model (tenant_id);

create index cloud_creds_tenant on cloud_creds_model (tenant_id);

create index docker_profile_tenant on docker_profile_model (tenant_id);

create index edge_tenant on edge_model (tenant_id);

create index edge_cert_tenant_edge on edge_cert_model (tenant_id, edge_id);

create index data_source_tenant_edge on data_source_model (tenant_id, edge_id);

create index data_stream_tenant on data_stream_model (tenant_id);

create index application_tenant on application_model (tenant_id);

create index sensor_tenant_edge on sensor_model (tenant_id, edge_id);

create index script_tenant on script_model (tenant_id);

create index user_tenant on user_model (tenant_id);

create index script_runtime_tenant on script_runtime_model (tenant_id);

create index app_status_tenant_application on application_status_model (tenant_id, application_id);

create index application_status_tenant_edge on application_status_model (tenant_id, edge_id);

create index edge_info_tenant_edge on edge_info_model (tenant_id, edge_id);

create index project_tenant on project_model (tenant_id);

create index tenant_rootca_tenant on tenant_rootca_model (tenant_id);

--changeset sherlock:29
-- Add external ID to link to my.nutanix.com tenant ID (adding some room in length)
ALTER TABLE tenant_model ADD COLUMN external_id varchar(100);

-- changeset sherlock:30
alter table data_stream_model ADD COLUMN end_point varchar(255);

--changeset sherlock:31
-- add UserProps table
create table if not exists user_props_model (
    user_id varchar(36) not null references user_model(id),
    tenant_id varchar(36) not null references tenant_model(id),
    props varchar(4096),
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table user_props_model drop constraint if exists user_props_model_user_id_fkey;
alter table user_props_model add constraint user_props_model_user_id_fkey FOREIGN KEY (user_id) REFERENCES user_model(id) ON DELETE CASCADE;
alter table user_props_model drop constraint if exists user_props_model_tenant_id_fkey;
alter table user_props_model add constraint user_props_model_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenant_model(id) ON DELETE CASCADE;
alter table user_props_model drop constraint if exists user_props_unique_user_id;
alter table user_props_model add constraint user_props_unique_user_id unique (user_id);

-- add TenantProps table
create table if not exists tenant_props_model (
    tenant_id varchar(36) not null references tenant_model(id),
    props varchar(4096),
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table tenant_props_model drop constraint if exists tenant_props_model_tenant_id_fkey;
alter table tenant_props_model add constraint tenant_props_model_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenant_model(id) ON DELETE CASCADE;
alter table tenant_props_model drop constraint if exists tenant_props_unique_tenant_id;
alter table tenant_props_model add constraint tenant_props_unique_tenant_id unique (tenant_id);

--changeset sherlock:32
-- Increase ScriptModel runtime_id length
ALTER TABLE script_model ALTER COLUMN runtime_id TYPE VARCHAR(64);

--changeset sherlock:33
-- Add kubeversion and osversion columns to edge_info table
alter table edge_info_model ADD COLUMN kube_version varchar(20);
alter table edge_info_model ADD COLUMN os_version varchar(20);

--changeset sherlock:34
-- Unique constraint for external ID
alter table tenant_model drop constraint if exists tenant_unique_external_id;
alter table tenant_model add constraint tenant_unique_external_id unique (external_id);

--changeset sherlock:35
-- Increase edge_info_model os_version length 
alter table edge_info_model ALTER COLUMN os_version TYPE varchar(64);

--changeset sherlock:36
-- Project unique name
alter table project_model drop constraint if exists project_model_unique_name;
alter table project_model add constraint project_model_unique_name unique (name, tenant_id);

--changeset sherlock:37
-- change some column type to text to not limit its size
alter table application_status_model ALTER COLUMN app_status TYPE text;
alter table application_model ALTER COLUMN yaml_data TYPE text;
alter table script_model ALTER COLUMN code TYPE text;

--changeset sherlock:38
-- Registration model for tenant pool service to hold registration code and config
create table if not exists tps_registration_model (
	id varchar(200) primary key,
	version bigserial not null,
	config jsonb,
	description varchar,
	state varchar(100),
	created_at timestamp not null,
    updated_at timestamp not null);

-- Tenant pool model for tenant pool service to hold tenant and other meta information
create table if not exists tps_tenant_pool_model (
    id varchar(100) primary key,
    version bigserial not null,
    registration_id varchar(200) not null references tps_registration_model(id),
    state varchar(100) not null,
    system_user varchar(200) not null unique,
    system_password varchar(200) not null,
    resources jsonb,
    created_at timestamp not null,
    updated_at timestamp not null);

-- Edge context model for for tenant pool service to hold edge information for each tenant
create table if not exists tps_edge_context_model (
    id varchar(1024) primary key,
    tenant_id varchar(100) not null references tps_tenant_pool_model(id) ON UPDATE CASCADE ON DELETE CASCADE,
    version bigserial not null,
    edge_id    varchar(100),
    state      varchar(100) not null,
    type       varchar(100) not null,
    created_at timestamp not null,
    updated_at timestamp not null);

--changeset sherlock:39
-- Add type field to edge_model
alter table edge_model add column type varchar(100);

--changeset sherlock:40
-- column to indicate trial or not
alter table tps_tenant_pool_model add column trial boolean default false;
--rollback alter table tps_tenant_pool_model drop column trial

--changeset sherlock:41
alter table edge_model add column short_id varchar(64);
alter table edge_model add constraint short_id_uniq_idx unique (short_id, tenant_id);
--rollback alter table edge_model drop column short_id

--changeset sherlock:42
-- Data source artifact model to keep records generated during the data source deployment
-- The records are immutable and can only be replaced
create table if not exists data_source_artifact_model (
	id serial primary key,
	tenant_id varchar(36) not null references tenant_model(id),
	data_source_id varchar(36) not null references data_source_model(id) ON DELETE CASCADE,
	data jsonb,
	version bigint not null,
	created_at timestamp without time zone default (now() at time zone 'utc'));

--changeset shelock:43
alter table data_source_model add column ifc_class varchar(20),
                              add column ifc_kind varchar(20),
                              add column ifc_protocol varchar(20),
                              add column ifc_img varchar(256),
                              add column ifc_project_id varchar(36),
                              add column ifc_driver_id varchar(36);
create table if not exists data_source_ifc_port_model(
    id serial primary key,
    data_source_id varchar(36) not null references data_source_model(id) ON DELETE CASCADE,
    name varchar(32) not null,
    port integer not null);

--changeset sherlock:44
create table if not exists machine_inference_model (
    id varchar(36) primary key,
    name varchar(200) not null,
    description varchar(200) not null,
    tenant_id varchar(36) not null references tenant_model(id) ON DELETE CASCADE,
    project_id varchar(64) not null references project_model(id) ON DELETE CASCADE,
    framework_type varchar(32) not null,
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);

create table if not exists machine_inference_version_model (
    id bigserial primary key,
    model_id varchar(36) not null references machine_inference_model(id) ON DELETE CASCADE,
    model_version int not null,
    s3_version varchar(1024) not null,
    description varchar(200) not null,
    model_size_bytes bigint not null);

--changeset sherlock:45
create index machine_inference_tenant on machine_inference_model (tenant_id);
create index machine_inference_name on machine_inference_model (name);
create index machine_inference_project on machine_inference_model (project_id);
create index machine_inference_framework on machine_inference_model (framework_type);
create index machine_inference_version_model_id on machine_inference_version_model (model_id);

--changeset sherlock:46
-- column to set the state of the application
-- default is null for backward compatibility
alter table application_model add column state varchar(100);
-- column to set the state of the data pipeline
-- default is null for backward compatibility
alter table data_stream_model add column state varchar(100);

--changeset shelock:47
-- add nullable assigned_at field to record when the tenant is assigned
alter table tps_tenant_pool_model add column assigned_at timestamp;

--changeset sherlock:48
-- audit log model to hold trial API requests
create table if not exists tps_audit_log_model (
    id serial primary key,
    tenant_id varchar(36),
    registration_id varchar(200),
    email varchar(200),
    actor varchar(200),
    action varchar(200) not null,
    response varchar(200) not null,
    description varchar(200),
    created_at timestamp without time zone default (now() at time zone 'utc'));

create index if not exists tps_audit_log_tenant_id on tps_audit_log_model(tenant_id);
create index if not exists tps_audit_log_registration_id on tps_audit_log_model(registration_id);
create index if not exists tps_audit_log_email on tps_audit_log_model(email);
create index if not exists tps_audit_log_actor on tps_audit_log_model(actor);
create index if not exists tps_audit_log_action on tps_audit_log_model(action);
create index if not exists tps_audit_log_response on tps_audit_log_model(response);

--changeset sherlock:49
-- indices to make edge inventory delta query more efficient
create index if not exists project_edge_model_project on project_edge_model(project_id);
create index if not exists project_edge_model_edge on project_edge_model(edge_id);
create index if not exists project_edge_selector_model_project on project_edge_selector_model(project_id);
create index if not exists application_edge_model_application on application_edge_model(application_id);
create index if not exists application_edge_model_edge on application_edge_model(edge_id);
create index if not exists application_edge_selector_model_application on application_edge_selector_model(application_id);
create index if not exists application_model_project on application_model(project_id);
create index if not exists data_stream_model_project on data_stream_model(project_id);
create index if not exists script_model_project on script_model(project_id);
create index if not exists script_runtime_model_project on script_runtime_model(project_id);
create index if not exists project_user_model_project on project_user_model(project_id);
create index if not exists project_cloud_creds_model_project on project_cloud_creds_model(project_id);
create index if not exists project_docker_profile_model_project on project_docker_profile_model(project_id);
create index if not exists edge_label_model_edge on edge_label_model(edge_id);

--changeset sherlock:50
-- record topic claims by data stream on a given data interface/source
create table if not exists data_stream_topic_claim (
    data_source_id varchar(36) REFERENCES data_source_model(id) ON DELETE CASCADE,
    data_stream_id varchar(36) REFERENCES data_stream_model(id) ON DELETE CASCADE,
    tenant_id varchar(36) not null references tenant_model(id) ON DELETE CASCADE,
    topic varchar(4096),
    CONSTRAINT pk PRIMARY KEY (data_source_id, topic),
    UNIQUE (data_stream_id));
-- rollback drop table if exists data_stream_topic_claim

-- changeset sherlock:51
-- change table name to be less specific to data stream
alter table data_stream_topic_claim rename to data_source_topic_claim
-- rollback alter table data_source_topic_claim rename to data_stream_topic_claim

-- changeset sherlock:52
-- add application_origin_model table
create table if not exists application_origin_model (
    id serial primary key,
    application_id varchar(36) not null references application_model(id) ON DELETE CASCADE,
    category_value_id integer not null references category_value_model(id),
    UNIQUE (application_id, category_value_id)
    );
-- rollback drop table if exits application_origin_model

-- changeset sherlock:53
-- add timestamp for machine_inference_version_model table
alter table machine_inference_version_model add column created_at timestamp, add column updated_at timestamp;
-- rollback alter table machine_inference_version_model drop column created_at, drop column updated_at;

--changeset sherlock:54
-- add application_id claim
alter table data_source_topic_claim add column application_id varchar(36) UNIQUE REFERENCES application_model(id) ON DELETE CASCADE;
-- rollback alter table data_source_topic_claim drop column application_id;

--changeset sherlock:55
-- Create the machine_inference_status_model to hold machine inference model status
create table if not exists machine_inference_status_model (
    tenant_id varchar(36) not null references tenant_model(id),
    edge_id varchar(36) not null references edge_model(id),
    model_id varchar(36) not null references machine_inference_model(id),
    version bigint not null,
    model_status text not null,
    created_at timestamp,
    updated_at timestamp,
    PRIMARY KEY(tenant_id, edge_id, model_id));
-- rollback drop table if exists machine_inference_status_model

--changeset sherlock:56
-- ON DELETE CASCADE for machine_inference_status_model foreign keys
alter table machine_inference_status_model drop constraint if exists machine_inference_status_model_tenant_id_fkey;
alter table machine_inference_status_model drop constraint if exists machine_inference_status_model_edge_id_fkey;
alter table machine_inference_status_model drop constraint if exists machine_inference_status_model_model_id_fkey;
alter table machine_inference_status_model add constraint machine_inference_status_model_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenant_model(id) ON DELETE CASCADE;
alter table machine_inference_status_model add constraint machine_inference_status_model_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
alter table machine_inference_status_model add constraint machine_inference_status_model_model_id_fkey FOREIGN KEY (model_id) REFERENCES machine_inference_model(id) ON DELETE CASCADE;
-- make ML model name unique per tenant for now
alter table machine_inference_model drop constraint if exists machine_inference_model_unique_name;
alter table machine_inference_model add constraint machine_inference_model_unique_name unique (name, tenant_id);

--changeset sherlock:57
-- Create data ifc topic association for application endpoints
create table if not exists application_endpoint_model (
    id serial primary key,
    application_id varchar(36) NOT NULL REFERENCES application_model(id) ON DELETE CASCADE,
    field_id integer NOT NULL REFERENCES data_source_field_model(id),
    tenant_id varchar(36) NOT NULL REFERENCES tenant_model(id),
    created_at timestamp NOT NULL,
    updated_at timestamp default(now() at time zone 'utc'),
    UNIQUE (application_id, field_id));
-- rollback drop table if exists application_endpoint_model

--changeset sherlock:58
-- Add field name to application endpoint model and backfill it for existing entries
alter table application_endpoint_model add column field_name varchar(200);
alter table application_endpoint_model add column data_source_id varchar(36) REFERENCES data_source_model(id) ;
update application_endpoint_model AS ae SET field_name=df.name, data_source_id=df.data_source_id FROM data_source_field_model AS df WHERE df.id = ae.field_id;
alter table application_endpoint_model alter column field_name SET NOT NULL;
alter table application_endpoint_model alter column data_source_id SET NOT NULL;
alter table application_endpoint_model drop column field_id
-- rollback alter table application_endpoint_model add column field_id integer REFERENCES data_source_field_model(id);
-- rollback update application_endpoint_model AS ae SET field_id=df.id FROM data_source_field_model AS df WHERE df.name = ae.field_name; 
-- rollback alter table application_endpoint_model alter column field_id SET NOT NULL;
-- rollback alter table application_endpoint_model drop column field_name;
-- rollback alter table application_endpoint_model drop column data_source_id;

--changeset sherlock:59
-- Add expiry timestamp field for every tenant claim record
alter table tps_tenant_pool_model add column expires_at timestamp;
--rollback alter table tps_tenant_pool_model drop column expires_at;

--changeset sherlock:60
-- Add on delete cascade to data_source_model edge_id column
alter table data_source_model drop constraint if exists data_source_model_edge_id_fkey;
alter table data_source_model add constraint data_source_model_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter table data_source_model drop constraint if exists data_source_model_edge_id_fkey;
-- rollback alter table data_source_model add constraint data_source_model_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES edge_model(id);

--changeset sherlock:61
-- fix user_role in project_user_model and start enforcing enum constraint
update project_user_model SET user_role = 'PROJECT_ADMIN' where user_role != 'PROJECT_ADMIN';
-- rollback

--changeset sherlock:62
-- create table user_public_key_model
create table if not exists user_public_key_model (
    id varchar(36) not null primary key references user_model(id) ON DELETE CASCADE,
    tenant_id varchar(36) NOT NULL REFERENCES tenant_model(id),
    public_key varchar(4096) not null,
    used_at timestamp not null,
    created_at timestamp not null,
    updated_at timestamp not null);
-- rollback drop table if exists user_public_key_model

--changeset sherlock:63
-- create table user_api_token_model
create table if not exists user_api_token_model (
    id varchar(36) not null primary key,
    tenant_id varchar(36) NOT NULL REFERENCES tenant_model(id),
    user_id varchar(36) NOT NULL REFERENCES user_model(id) ON DELETE CASCADE,
    active boolean not null,
    used_at timestamp not null,
    created_at timestamp not null,
    updated_at timestamp not null);
-- rollback drop table if exists user_api_token_model

--changeset sherlock:64
alter table cloud_creds_model add column az_credential VARCHAR(1024);
--rollback alter table cloud_creds_model drop column az_credential;

--changeset sherlock:65
-- create table edge_cluster and edge_device
create table if not exists edge_cluster_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    short_id varchar(64),
    connected boolean DEFAULT false,
    type varchar(100),
    created_at timestamp not null,
    updated_at timestamp not null,
    constraint edge_cluster_model_unique_name unique (name, tenant_id),
    constraint edge_cluster_short_id_uniq_idx unique (short_id, tenant_id)
);
create index edge_cluster_tenant on edge_cluster_model (tenant_id);
-- rollback drop table if exists edge_cluster_model

--changeset sherlock:66
create table if not exists edge_device_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    edge_cluster_id varchar(36) not null,
    serial_number varchar(200) not null,
    ip_address varchar(20),
    gateway varchar(20),
    subnet varchar(20),
    created_at timestamp not null,
    updated_at timestamp not null,
    constraint edge_device_model_unique_name unique (name, tenant_id),
    constraint edge_device_model_unique_serial unique (serial_number)
);
create index edge_device_tenant on edge_device_model (tenant_id);
-- rollback drop table if exists edge_device_model

--changeset sherlock:67
alter table data_stream_model add column az_stream_type VARCHAR(20);
-- rollback alter table data_stream_model drop column az_stream_type

--changeset sherlock:68
alter table edge_device_model add constraint edge_device_model_edge_cluster_id_fkey FOREIGN KEY (edge_cluster_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter table edge_device_model ADD COLUMN is_onboarded boolean not null DEFAULT false,
    ADD COLUMN is_bootstrap_master boolean,
    ADD COLUMN ssh_public_key varchar(500);
alter table edge_device_model add constraint edge_device_model_unique_cluster_bootstrap_master unique (edge_cluster_id, is_bootstrap_master);
-- rollback alter table edge_device_model drop constraint if exists edge_device_model_edge_cluster_id_fkey
-- rollback alter table edge_device_model drop column is_bootstrap_master
-- rollback alter table edge_device_model drop column is_onboarded
-- rollback alter table edge_device_model drop column ssh_public_key

--changeset sherlock:69
alter table edge_info_model ADD COLUMN edge_artifact jsonb;
-- rollback alter table edge_info_model drop column edge_artifact

--changeset sherlock:70
-- migrate the data
create index edge_device_cluster on edge_device_model (edge_cluster_id);

INSERT INTO  edge_cluster_model (
    id, version, tenant_id, name, description, created_at, updated_at, connected , type , short_id) 
    SELECT id, version, tenant_id, name, description, created_at, updated_at, connected, type, short_id 
    FROM edge_model;

INSERT INTO edge_device_model (
    id, version, tenant_id, name, description, created_at, updated_at, edge_cluster_id, serial_number, ip_address, gateway, subnet)  
    SELECT id, version, tenant_id, name, description, created_at, updated_at, id as edge_cluster_id, serial_number, ip_address, gateway, subnet
    FROM edge_model;


-- Migrate data from edge_model to edge_cluster and edge_device model
alter TABLE "application_edge_model" drop CONSTRAINT "application_edge_model_edge_id_fkey";
alter TABLE "application_status_model" drop CONSTRAINT "application_status_model_edge_id_fkey";
alter TABLE "data_source_model" drop CONSTRAINT "data_source_model_edge_id_fkey";
alter TABLE "edge_cert_model" drop CONSTRAINT "edge_cert_model_edge_id_fkey";
alter TABLE "edge_info_model" drop CONSTRAINT "edge_info_model_edge_id_fkey";
alter TABLE "edge_label_model" drop CONSTRAINT "edge_label_model_edge_id_fkey";
alter TABLE "log_model" drop CONSTRAINT "log_model_edge_id_fkey";
alter TABLE "machine_inference_status_model" drop CONSTRAINT "machine_inference_status_model_edge_id_fkey";
alter TABLE "project_edge_model" drop CONSTRAINT "project_edge_model_edge_id_fkey";
alter TABLE "sensor_model" drop CONSTRAINT "sensor_model_edge_id_fkey";
alter TABLE "edge_model" drop CONSTRAINT "edge_model_tenant_id_fkey";

alter TABLE "application_edge_model" add CONSTRAINT "application_edge_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "application_status_model" add CONSTRAINT "application_status_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "data_source_model" add CONSTRAINT "data_source_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "edge_cert_model" add CONSTRAINT "edge_cert_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "log_model" add CONSTRAINT "log_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "machine_inference_status_model" add CONSTRAINT "machine_inference_status_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "project_edge_model" add CONSTRAINT "project_edge_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "sensor_model" add CONSTRAINT "sensor_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
alter TABLE "edge_label_model" add CONSTRAINT "edge_label_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;

alter TABLE "edge_info_model" add CONSTRAINT "edge_info_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_device_model(id) ON DELETE CASCADE;

-- rollback alter TABLE "edge_info_model" drop CONSTRAINT "edge_info_model_edge_id_fkey";
-- rollback alter TABLE "edge_label_model" drop CONSTRAINT "edge_label_model_edge_id_fkey";
-- rollback alter TABLE "sensor_model" drop CONSTRAINT "sensor_model_edge_id_fkey";
-- rollback alter TABLE "project_edge_model" drop CONSTRAINT "project_edge_model_edge_id_fkey";
-- rollback alter TABLE "machine_inference_status_model" drop CONSTRAINT "machine_inference_status_model_edge_id_fkey";
-- rollback alter TABLE "log_model" drop CONSTRAINT "log_model_edge_id_fkey";
-- rollback alter TABLE "edge_cert_model" drop CONSTRAINT "edge_cert_model_edge_id_fkey";
-- rollback alter TABLE "data_source_model" drop CONSTRAINT "data_source_model_edge_id_fkey";
-- rollback alter TABLE "application_status_model" drop CONSTRAINT "application_status_model_edge_id_fkey";
-- rollback alter TABLE "application_edge_model" drop CONSTRAINT "application_edge_model_edge_id_fkey";

-- rollback alter TABLE "edge_model" add CONSTRAINT "edge_model_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenant_model(id);
-- rollback alter TABLE "sensor_model" add CONSTRAINT "sensor_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "project_edge_model" add CONSTRAINT "project_edge_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "machine_inference_status_model" add CONSTRAINT "machine_inference_status_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "log_model" add CONSTRAINT "log_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "edge_label_model" add CONSTRAINT "edge_label_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "edge_info_model" add CONSTRAINT "edge_info_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "edge_cert_model" add CONSTRAINT "edge_cert_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "data_source_model" add CONSTRAINT "data_source_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "application_status_model" add CONSTRAINT "application_status_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;
-- rollback alter TABLE "application_edge_model" add CONSTRAINT "application_edge_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_model(id) ON DELETE CASCADE;

-- rollback delete FROM edge_cluster_model;
-- rollback delete FROM edge_device_model;
-- rollback DROP INDEX edge_device_cluster;

--changeset sherlock:71
create index if not exists application_endpoint_app on application_endpoint_model (application_id);
create index if not exists application_origin_app on application_origin_model (application_id);

-- rollback DROP INDEX application_endpoint_app;
-- rollback DROP INDEX application_origin_app;

--changeset sherlock:72
create table if not exists project_service_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    service_manifest varchar(2000),
    project_id varchar(64) not null references project_model(id) ON DELETE CASCADE,
    created_at timestamp not null,
    updated_at timestamp not null);

-- rollback drop table if exists project_service_model;

--changeset sherlock:73
alter table project_service_model drop constraint if exists project_service_model_unique_name;
alter table project_service_model add constraint project_service_model_unique_name unique (name, tenant_id, project_id);

-- rollback alter table project_service_model drop constraint project_service_model_unique_name;

--changeset sherlock:74
alter table edge_cluster_model add column virtual_ip varchar (20);

-- rollback alter table edge_cluster_model drop column virtual_ip;

create table if not exists edge_device_info_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    device_id varchar(36) not null references edge_device_model(id) ON DELETE CASCADE,
    edge_version varchar(20),
    edge_build_num varchar(20),
    kube_version varchar(20),
    os_version varchar(64),
    artifacts jsonb,
    num_cpu varchar(20),
    total_memory_kb varchar(20),
    total_storage_kb varchar(20),
    gpu_info varchar(20),
    cpu_usage varchar(20),
    memory_free_kb varchar(20),
    storage_free_kb varchar(20),
    gpu_usage varchar(20),
    healthy boolean,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table edge_device_info_model drop constraint if exists edge_device_info_unique_device;
alter table edge_device_info_model add constraint edge_device_info_unique_device unique (tenant_id, device_id);

-- migrate edge_info_model records, healthy is not used for edge_info_model
INSERT INTO edge_device_info_model (id, version, tenant_id, device_id, edge_version, edge_build_num, kube_version, os_version, artifacts, num_cpu, total_memory_kb, total_storage_kb, gpu_info, cpu_usage, memory_free_kb, storage_free_kb, gpu_usage, created_at, updated_at, healthy)
    SELECT id, version, tenant_id, id as device_id, edge_version, edge_build_num, kube_version, os_version, edge_artifact as artifacts, num_cpu, total_memory_kb, total_storage_kb, gpu_info, cpu_usage, memory_free_kb, storage_free_kb, gpu_usage, created_at, updated_at, false as healthy
    FROM edge_info_model ON CONFLICT DO NOTHING;

-- rollback drop table if exists edge_device_info_model

--changeset sherlock:75
alter table edge_device_info_model add column health_bits jsonb,
    drop column healthy;

-- rollback alter table edge_device_info_model drop column health_bits, add column healthy boolean;

--changeset sherlock:76
create table if not exists service_domain_info_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    edge_cluster_id varchar(36) not null references edge_cluster_model(id) ON DELETE CASCADE,
    artifacts jsonb,
    created_at timestamp not null,
    updated_at timestamp not null);

alter table service_domain_info_model drop constraint if exists service_domain_info_unique_domain;
alter table service_domain_info_model add constraint service_domain_info_unique_domain unique (tenant_id, edge_cluster_id);

-- rollback drop table if exists service_domain_info_model;

--changeset sherlock:77

create table if not exists edge_log_collect_model (
    id varchar(36) primary key,
    version bigint not null,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    project_id varchar(36) null references project_model(id) ON DELETE CASCADE,
    cloud_creds_id varchar(36) references cloud_creds_model(id),
    sources jsonb,
    code varchar(30720) null,
    state varchar(100) not null,
    type varchar(100) not null,
    created_at timestamp not null,
    updated_at timestamp not null
);

alter table edge_log_collect_model drop constraint if exists edge_log_collect_model_unique_name;
alter table edge_log_collect_model add constraint edge_log_collect_model_unique_name unique (name, tenant_id);
-- rollback drop table if exists edge_log_collect_model;

--changeset sherlock:78

alter table cloud_creds_model alter column aws_credential type varchar(4096);

-- rollback alter table cloud_creds_model alter column aws_credential type varchar(500);

--changeset sherlock:79
alter table edge_cluster_model add column profile text;
-- rollback alter table edge_cluster_model drop column profile;

--changeset sherlock:80
alter table edge_log_collect_model add column dest varchar(512);
-- rollback alter table edge_log_collect_model drop column dest;

--changeset sherlock:81
-- model for software download or upgrade batch
create table if not exists software_update_batch_model (
    id varchar(36) primary key,
    tenant_id varchar(36) not null references tenant_model(id),
    type varchar(36) not null,
    release varchar(36) not null,
    created_at timestamp not null,
    updated_at timestamp default(now() at time zone 'utc')
);

-- model to hold service domain states for each batch
create table if not exists software_update_service_domain_model (
	id serial primary key,
	tenant_id varchar(36) not null references tenant_model(id),
	batch_id varchar(36) not null references software_update_batch_model(id) on delete cascade,
	svc_domain_id varchar(36) not null references edge_cluster_model(id) on delete cascade,
	state varchar(36) not null,
	progress int not null,
	eta int not null,
	failure_reason varchar,
	created_at timestamp not null,
	updated_at timestamp default(now() at time zone 'utc'),
	state_updated_at timestamp not null
);

-- model to hold service demains with the downlaoded release version
create table if not exists software_update_downloaded_model (
	id serial primary key,
	tenant_id varchar(36) not null references tenant_model(id),
	svc_domain_id varchar(36) not null unique references edge_cluster_model(id) on delete cascade,
	batch_id varchar(36) not null references software_update_batch_model(id),
	release varchar(36) not null,
	created_at timestamp default(now() at time zone 'utc')
);

-- rollback drop table if exists software_update_downloaded_model;
-- rollback drop table if exists software_update_service_domain_model;
-- rollback drop table if exists software_update_batch_model;

--changeset sherlock:82
update edge_log_collect_model set dest = '' where dest is null;
alter table edge_log_collect_model alter column dest set not null;
-- rollback alter table edge_log_collect_model alter column dest drop not null;

--changeset sherlock:83
alter table tenant_model ADD COLUMN IF NOT EXISTS profile text;
-- rollback alter table tenant_model drop column profile;

--changeset sherlock:84
alter table application_model ADD COLUMN IF NOT EXISTS only_pre_pull_on_update boolean;
-- rollback alter table application_model drop column only_pre_pull_on_update;

--changeset sherlock:85
alter table application_model alter column only_pre_pull_on_update set default false;
update application_model set only_pre_pull_on_update=false;
alter table application_model alter column only_pre_pull_on_update set not null;
-- rollback alter table application_model alter column only_pre_pull_on_update drop not null;
-- rollback alter table application_model alter column only_pre_pull_on_update drop DEFAULT;

--changeset sherlock:86
alter table edge_log_collect_model add column aws_cloudwatch varchar(4096);
alter table edge_log_collect_model add column aws_kinesis varchar(4096);
alter table edge_log_collect_model add column gcp_stackdriver varchar(4096);
update edge_log_collect_model set aws_cloudwatch = '{"dest": "' || dest  || '", "stream": "' || name || '", "group: "' || name || '"}';
update edge_log_collect_model set dest = 'CLOUDWATCH';
-- rollback alter table edge_log_collect_model drop column aws_cloudwatch;
-- rollback alter table edge_log_collect_model drop column aws_kinesis;
-- rollback alter table edge_log_collect_model drop column gcp_stackdriver;

--changeset sherlock:87
-- Make batch record deletable without deleting the reference from downloaded table
alter table software_update_downloaded_model drop constraint if exists software_update_downloaded_model_batch_id_fkey;
-- rollback alter table software_update_downloaded_model add constraint software_update_downloaded_model_batch_id_fkey foreign key (batch_id) references software_update_batch_model(id);

--changeset sherlock:88
alter table application_model ALTER COLUMN description TYPE varchar(512);
-- rollback alter application_model ALTER COLUMN description TYPE varchar(200);

--changeset sherlock:89
alter table project_model ADD COLUMN IF NOT EXISTS privileged boolean;
alter table application_model ADD COLUMN IF NOT EXISTS packaging_type VARCHAR(36);
alter table application_model ADD COLUMN IF NOT EXISTS helm_metadata jsonb;
-- rollback alter table project_model DROP COLUMN privileged;
-- rollback alter table application_model DROP COLUMN packaging_type;
-- rollback alter table application_model DROP COLUMN helm_metadata;

--changeset sherlock:90
alter table edge_cluster_model ADD COLUMN IF NOT EXISTS env text;
-- rollback alter table edge_cluster_model drop column env;

--changeset sherlock:91
alter table application_edge_model ADD COLUMN IF NOT EXISTS state VARCHAR(20) DEFAULT 'DEPLOY'
-- rollback alter table application_edge_model DROP COLUMN IF EXISTS state;

--changeset sherlock:92
-- Create the StorageProfile table to hold the storage profile model objects
create table if not exists storage_profile_model (
    id varchar(36) primary key,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    type varchar(20) not null,
    aos_config varchar(4096),
    ebs_config varchar(4096),
    vsphere_config varchar(4096),
    iflag_encrypted boolean,
    created_at timestamp not null,
    updated_at timestamp not null);
-- Create the SvcDomainStorageProfileModel to hold svc domain <-> storage profile relation
create table if not exists svcdomain_storage_profile_model (
    id serial primary key,
    svc_domain_id varchar(36) not null references edge_cluster_model(id) ON DELETE CASCADE,
    storage_profile_id varchar(36) not null references storage_profile_model(id) ON DELETE CASCADE);
-- rollback drop table if exists svcdomain_storage_profile_model
-- rollback drop table if exists storage_profile_model

--changeset sherlock:93
alter table edge_device_model ADD COLUMN IF NOT EXISTS role text not null DEFAULT '{"master":true,"worker":true}';
-- rollback alter table edge_device_model drop column role;

--changeset sherlock:94
create table if not exists service_class_model (
    id varchar(36) primary key,
    name varchar(200) not null,
    description varchar(200),
    type varchar(100) not null,
    svc_version varchar(36) not null,
    scope varchar(100) not null,
    state varchar(100) not null,
    min_svc_domain_version varchar(36) not null,
    bindable boolean,
    svc_instance_create_schema jsonb,
    svc_instance_update_schema jsonb,
    svc_binding_create_schema jsonb,
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);
-- rollback drop table if exists service_class_model;

alter table service_class_model drop constraint if exists service_class_unique_name;
alter table service_class_model add constraint service_class_unique_name unique (name);
-- rollback alter table service_class_model drop constraint if exists service_class_unique_name;

alter table service_class_model drop constraint if exists service_class_unique_type;
alter table service_class_model add constraint service_class_unique_stype unique (type);
-- rollback alter table service_class_model drop constraint if exists service_class_unique_stype;

create table if not exists service_instance_model (
    id varchar(36) primary key,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    svc_class_id varchar(36) not null references service_class_model(id),
    svc_domain_scope_id varchar(36) references edge_cluster_model(id) ON DELETE CASCADE,
    project_scope_id varchar(36) references project_model(id) ON DELETE CASCADE,
    parameters jsonb,
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);
-- rollback drop table if exists service_instance_model;

-- allow only one instance for a scope ID and class ID
alter table service_instance_model drop constraint if exists service_instance_unique_instance;
-- NULL values do not take part in constraint
alter table service_instance_model add constraint service_instance_unique_instance unique (tenant_id, svc_class_id, svc_domain_scope_id, project_scope_id);
-- rollback alter table service_instance_model drop constraint if exists service_instance_unique_instance;

alter table service_instance_model drop constraint if exists service_instance_unique_name;
-- NULL values do not take part in constraint
alter table service_instance_model add constraint service_instance_unique_name unique (tenant_id, name, svc_domain_scope_id, project_scope_id);
-- rollback alter table service_instance_model drop constraint if exists service_instance_unique_name;

create table if not exists service_binding_model (
    id varchar(36) primary key,
    tenant_id varchar(36) not null references tenant_model(id),
    name varchar(200) not null,
    description varchar(200),
    svc_class_id varchar(36) not null references service_class_model(id),
    resource_type varchar(100),
    svc_domain_resource_id varchar(36) references edge_cluster_model(id) ON DELETE CASCADE,
    project_resource_id varchar(64) references project_model(id) ON DELETE CASCADE,
    parameters jsonb,
    version bigint not null,
    created_at timestamp not null,
    updated_at timestamp not null);
-- rollback drop table if exists service_binding_model;

alter table service_binding_model drop constraint if exists service_binding_unique_name;
alter table service_binding_model add constraint service_binding_unique_name unique (tenant_id, name, svc_class_id);
-- rollback alter table service_binding_model drop constraint if exists service_binding_unique_name;

--changeset sherlock:95
alter table service_class_model add column if not exists tags jsonb;
-- rollback alter table service_class_model drop column if exists tags;

--changeset sherlock:96
CREATE INDEX IF NOT EXISTS category_value_category_id ON category_value_model(category_id);
CREATE INDEX IF NOT EXISTS data_source_field_data_source_id ON data_source_field_model(data_source_id);
CREATE INDEX IF NOT EXISTS application_endpoint_data_source_id ON application_endpoint_model(data_source_id);
-- rollback DROP INDEX category_value_category_id;
-- rollback DROP INDEX data_source_field_data_source_id;
-- rollback DROP INDEX application_endpoint_data_source_id;

--changeset sherlock:97
alter table storage_profile_model ADD COLUMN IF NOT EXISTS isdefault boolean DEFAULT FALSE

--changeset sherlock:98
alter table application_model drop constraint if exists application_model_unique_name;
-- rollback alter table application_model add constraint application_model_unique_name unique (name, tenant_id);
alter table application_model add constraint application_model_unique_name unique (name, tenant_id, project_id);
-- rollback alter table application_model drop constraint if exists application_model_unique_name;

alter table data_stream_model drop constraint if exists data_stream_model_unique_name;
-- rollback alter table data_stream_model add constraint data_stream_model_unique_name unique (tenant_id, name);
alter table data_stream_model add constraint data_stream_model_unique_name unique (tenant_id, name, project_id);
-- alter table data_stream_model drop constraint if exists data_stream_model_unique_name;

-- Service Class description can be very long
alter table service_class_model alter column description type varchar(1024);

--changeset sherlock:99
create table if not exists http_service_proxy_model (
    id varchar(36) primary key,
    tenant_id varchar(36) not null references tenant_model(id) ON DELETE CASCADE,
    edge_cluster_id varchar(36) not null references edge_cluster_model(id) ON DELETE CASCADE,
    name varchar(200) not null,
    type varchar(20) not null,
    project_id varchar(64) references project_model(id) ON DELETE CASCADE,
    service_name varchar(64) not null,
    service_port integer not null,
    service_namespace varchar(200),
    duration varchar(64) not null,
    username varchar(32),
    password varchar(64),
    hostname varchar(64) not null,
    hostport integer not null,
    version bigint not null,
    public_key varchar(4096),
    expires_at timestamp not null,
    created_at timestamp not null,
    updated_at timestamp not null);
alter table http_service_proxy_model drop constraint if exists http_service_proxy_unique_name;
alter table http_service_proxy_model add constraint http_service_proxy_unique_name unique (tenant_id, edge_cluster_id, name);
-- rollback drop table if exists http_service_proxy_model;

--changeset sherlock:100
-- rollback drop table if exists kubernetes_cluster_info_model;
alter table tenant_model add column deleted_at timestamp;
-- rollback alter table tenant_model drop column if exists deleted_at;
alter table tenant_model add column created_by varchar(100);
-- rollback alter table tenant_model drop column if exists created_by;

--changeset sherlock:101
create table if not exists kubernetes_cluster_info_model (
    id varchar(36) primary key references edge_cluster_model(id) ON DELETE CASCADE,
    tenant_id varchar(36) not null references tenant_model(id),
    chart_version varchar(50),
    kube_version varchar(50),
    onboarded boolean);
-- rollback drop table if exists kubernetes_cluster_info_model;

--changeset sherlock:102
create table if not exists service_domain_viewonly_user_model (
    edge_cluster_id varchar(36) not null references edge_cluster_model(id) ON DELETE CASCADE,
    user_id varchar(36) not null references user_model(id) ON DELETE CASCADE,
    created_at timestamp not null,
    unique(edge_cluster_id, user_id)
);
create index if not exists service_domain_viewonly_user_model_service_domain on service_domain_viewonly_user_model(edge_cluster_id);
create index if not exists service_domain_viewonly_user_model_user on service_domain_viewonly_user_model(user_id);
-- rollback drop index service_domain_viewonly_user_model_service_domain;
-- rollback drop index service_domain_viewonly_user_model_user;
-- rollback drop table if exists service_domain_viewonly_user_model;

--changeset sherlock:103
alter table script_model drop constraint if exists script_model_unique_name;
-- rollback alter table script_model add constraint script_model_unique_name unique (name, tenant_id);
alter table script_model add constraint script_model_unique_name unique (name, tenant_id, project_id);
-- rollback alter table script_model drop constraint if exists script_model_unique_name;

alter table script_runtime_model drop constraint if exists script_runtime_model_unique_name;
-- rollback alter table script_runtime_model add constraint script_runtime_model_unique_name unique (name, tenant_id);
alter table script_runtime_model add constraint script_runtime_model_unique_name unique (name, tenant_id, project_id);
-- rollback alter table script_runtime_model drop constraint if exists script_runtime_model_unique_name;

--changeset sherlock:104
create table if not exists data_driver_class_model(
    id                     varchar(36)    primary key,
    version                bigint         not null,
    tenant_id              varchar(36)    not null references tenant_model (id),
    name                   varchar(200)   not null,
    description            varchar(200),

    driver_version         varchar(20)    not null,
    min_svc_domain_version varchar(36)    not null,

    type                   varchar(100)   not null,

    yaml_data              varchar(30720) not null,
    static_schema          jsonb,
    dynamic_schema         jsonb,

    created_at             timestamp      not null,
    updated_at             timestamp      not null);

create index data_driver_class_tenant on data_driver_class_model (tenant_id);

alter table data_driver_class_model
    drop constraint if exists data_driver_class_name_version_unique;
alter table data_driver_class_model
    add constraint data_driver_class_name_version_unique unique (tenant_id, name, driver_version);
-- rollback drop index data_driver_class_tenant;
-- rollback drop table if exists data_driver_class_model;

--changeset sherlock:105
create table if not exists data_driver_instance_model(
    id                   varchar(36)  primary key,
    version              bigint       not null,
    tenant_id            varchar(36)  not null references tenant_model (id),
    name                 varchar(200) not null,
    description          varchar(200),
    data_driver_class_id varchar(36)  not null references data_driver_class_model (id),
    project_id           varchar(36)  not null references project_model (id) ON DELETE CASCADE,
    parameters           jsonb,
    created_at           timestamp    not null,
    updated_at           timestamp    not null);

alter table data_driver_instance_model
    add constraint data_driver_instance_name_unique unique (tenant_id, project_id, name);
create index data_driver_instance_tenant on data_driver_instance_model (tenant_id);
create index data_driver_instance_project on data_driver_instance_model (project_id);
-- rollback drop index data_driver_instance_tenant;
-- rollback drop index data_driver_instance_project;
-- rollback drop table if exists data_driver_instance_model;

--changeset sherlock:106
create table if not exists data_driver_params_model(
    id                   varchar(36) primary key,
    version              bigint      not null,
    tenant_id            varchar(36) not null references tenant_model (id),
    instance_id          varchar(36) not null references data_driver_instance_model (id),
    name                 varchar(200) not null,
    description          varchar(200),
    parameters           jsonb,
    created_at           timestamp   not null,
    updated_at           timestamp   not null);
alter table data_driver_params_model
    add constraint data_driver_params_model_name_unique unique (instance_id, name);
create index data_driver_params_tenant on data_driver_params_model (tenant_id);
create index data_driver_params_instance on data_driver_params_model (instance_id);
-- rollback drop index data_driver_params_tenant;
-- rollback drop index data_driver_instance_model;
-- rollback drop table if exists data_driver_params_model;

--changeset sherlock:107
create table if not exists data_driver_label_model(
    id                bigserial   primary key,
    params_id         varchar(36) not null references data_driver_params_model (id) ON DELETE CASCADE,
    category_value_id integer     not null references category_value_model (id) ON DELETE CASCADE);
alter table data_driver_label_model
    drop constraint if exists data_driver_label_unique_value;
alter table data_driver_label_model
    add constraint data_driver_label_unique_value unique (params_id, category_value_id);
-- rollback drop table if exists data_driver_label_model;

create index data_driver_label_params on data_driver_label_model (params_id);
-- rollback drop index data_driver_label_params;

create table if not exists data_driver_edge_model(
    id        bigserial   primary key,
    params_id varchar(36) not null references data_driver_params_model (id) ON DELETE CASCADE,
    edge_id   varchar(36) not null references edge_model (id) ON DELETE CASCADE);
alter table data_driver_edge_model
    drop constraint if exists data_driver_edge_unique_pair;
alter table data_driver_edge_model
    add constraint data_driver_edge_unique_pair unique (params_id, edge_id);
-- rollback drop table if exists data_driver_edge_model;

create index data_driver_edge_params on data_driver_edge_model (params_id);
-- rollback drop index data_driver_edge_params;

create table if not exists data_driver_edge_selector_model (
    id                bigserial   primary key,
    params_id         varchar(36) not null references data_driver_params_model (id) ON DELETE CASCADE,
    category_value_id integer     not null references category_value_model (id) ON DELETE CASCADE);
alter table data_driver_edge_selector_model
    drop constraint if exists data_driver_edge_selector_unique_value;
alter table data_driver_edge_selector_model
    add constraint data_driver_edge_selector_unique_value unique (params_id, category_value_id);
-- rollback drop table if exists data_driver_edge_selector_model;

create index data_driver_edge_selector_params on data_driver_edge_selector_model (params_id);
-- rollback drop index data_driver_edge_selector_params;

--changeset sherlock:108
alter table data_driver_class_model drop column if exists stream_schema;
alter table data_driver_class_model add column stream_schema jsonb;
-- rollback alter table data_stream_model drop column if exists stream_schema;

alter table data_driver_params_model drop column if exists direction;
alter table data_driver_params_model add column direction varchar(200);
-- rollback alter table data_driver_params_model drop column if exists direction;

alter table data_driver_params_model drop column if exists type;
alter table data_driver_params_model add column type varchar(32);
-- rollback alter table data_driver_params_model drop column if exists type;

--changeset sherlock:109
alter table data_driver_edge_model ADD COLUMN IF NOT EXISTS state VARCHAR(20) DEFAULT 'DEPLOY';
alter TABLE data_driver_edge_model DROP CONSTRAINT "data_driver_edge_model_edge_id_fkey";
alter table data_driver_edge_model ADD CONSTRAINT "data_driver_edge_model_edge_id_fkey" FOREIGN KEY (edge_id) REFERENCES edge_cluster_model(id) ON DELETE CASCADE;
-- rollback alter table data_driver_edge_model DROP COLUMN IF EXISTS state;

--changeset sherlock:110
alter table data_driver_params_model drop constraint data_driver_params_model_name_unique;
alter table data_driver_params_model add constraint data_driver_params_model_name_unique unique (instance_id, name, type);
-- rollback alter table data_driver_params_model drop constraint data_driver_params_model_name_unique;
-- rollback alter table data_driver_params_model add constraint data_driver_params_model_name_unique unique (instance_id, name);
