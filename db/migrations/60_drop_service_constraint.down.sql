alter table cd_pipeline_third_party_service add constraint service_fk foreign key (service_id) references third_party_service (id);