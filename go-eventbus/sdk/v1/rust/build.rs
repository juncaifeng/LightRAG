fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_dir = std::path::PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").unwrap())
        .parent().unwrap().parent().unwrap().parent().unwrap()
        .join("proto");

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile_protos(
            &[
                proto_dir.join("lightrag_eventbus.proto").to_str().unwrap(),
                proto_dir.join("topics").join("insert.proto").to_str().unwrap(),
                proto_dir.join("topics").join("query.proto").to_str().unwrap(),
            ],
            &[
                proto_dir.to_str().unwrap(),
                proto_dir.join("topics").to_str().unwrap(),
            ],
        )?;
    Ok(())
}
