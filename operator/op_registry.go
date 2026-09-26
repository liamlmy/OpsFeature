package operator

import "github.com/liamlmy/OpsFeature/feature"

func init() {
	feature.Register("Direct", func() feature.Operator { return &Direct{} })
	feature.Register("Seq", func() feature.Operator { return &Seq{} })
	feature.Register("Seq_field", func() feature.Operator { return &SeqField{} })
	feature.Register("Seq_emb", func() feature.Operator { return &SeqEmb{} })
	feature.Register("Bucket", func() feature.Operator { return &Bucket{} })
	feature.Register("Bucket_truncate", func() feature.Operator { return &BucketTruncate{} })
	feature.Register("Combine", newCombine(2))
	feature.Register("Combine_multi", newCombine(3))
	feature.Register("Arithmetic", func() feature.Operator { return &Arithmetic{} })
	feature.Register("Scale", func() feature.Operator { return &Scale{} })
}
