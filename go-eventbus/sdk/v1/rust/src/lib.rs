pub mod lightrag {
    pub mod eventbus {
        pub mod v1 {
            tonic::include_proto!("lightrag.eventbus.v1");
        }
        pub mod topics {
            pub mod v1 {
                tonic::include_proto!("lightrag.eventbus.topics.v1");
            }
        }
    }
}

pub use lightrag::eventbus::v1::*;
pub mod topics {
    pub use crate::lightrag::eventbus::topics::v1::*;
}
