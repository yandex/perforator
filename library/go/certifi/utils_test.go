package certifi_test

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
)

func getCAsFromPool(certPool *x509.CertPool) ([]string, error) {
	subjects := certPool.Subjects()
	result := make([]string, len(subjects))
	for i, rawSubj := range subjects {
		var rdns pkix.RDNSequence
		if _, err := asn1.Unmarshal(rawSubj, &rdns); err != nil {
			return nil, err
		}

		name := pkix.Name{}
		name.FillFromRDNSequence(&rdns)
		result[i] = name.CommonName
	}
	return result, nil
}
