fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_dir = std::path::PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").unwrap())
        .parent().unwrap().parent().unwrap().parent().unwrap()
        .join("proto");

    let topic_dir = proto_dir.join("topics");

    // compile_protos: only direct entry points (not imported dependencies).
    // rag/insert.proto imports index/builder.proto, so builder.proto doesn't need to be listed here.
    // index/retriever.proto is standalone (not imported by anyone).
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile_protos(
            &[
                proto_dir.join("lightrag_eventbus.proto").to_str().unwrap(),
                topic_dir.join("rag").join("insert.proto").to_str().unwrap(),
                topic_dir.join("rag").join("query.proto").to_str().unwrap(),
                topic_dir.join("index").join("retriever.proto").to_str().unwrap(),
            ],
            &[
                proto_dir.to_str().unwrap(),
                topic_dir.to_str().unwrap(),
            ],
        )?;
    Ok(())
}
