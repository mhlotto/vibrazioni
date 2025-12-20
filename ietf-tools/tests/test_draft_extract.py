import importlib.util
import sys
from pathlib import Path
import unittest


def load_module(name, rel_path):
    base_dir = Path(__file__).resolve().parents[1]
    path = base_dir / rel_path
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


extract = load_module("draft_extract", "ietf-draft-extract.py")

EXTRACT_CASES = {
    "testdata/draft-ietf-scitt-architecture-22.txt": [
        {
            "query": "3",
            "expected_output": """3.  Terminology

   The terms defined in this section have special meaning in the context
   of Supply Chain Integrity, Transparency, and Trust, and are used
   throughout this document.

   This document has been developed in coordination with the COSE, OAUTH
   and RATS WG and uses terminology common to these working groups as
   much as possible.

   When used in text, the corresponding terms are capitalized.  To
   ensure readability, only a core set of terms is included in this
   section.

   The terms "header", "payload", and "to-be-signed bytes" are defined
   in [STD96].




Birkholz, et al.          Expires 13 April 2026                [Page 10]


Internet-Draft             SCITT Architecture               October 2025


   The term "claim" is defined in [RFC8392].

   Append-only Log:  a Statement Sequence comprising the entire
      registration history of the Transparency Service.  To make the
      Append-only property verifiable and transparent, the Transparency
      Service defines how Signed Statements are made available to
      Auditors.

   Artifact:  a physical or non-physical item that is moving along a
      supply chain.

   Auditor:  an entity that checks the correctness and consistency of
      all Transparent Statements, or the transparent Statement Sequence,
      issued by a Transparency Service.  An Auditor is an example of a
      specialized Relying Party.

   Client:  an application making protected Transparency Service
      resource requests on behalf of the resource owner and with its
      authorization.

   Envelope:  metadata, created by the Issuer to produce a Signed
      Statement.  The Envelope contains the identity of the Issuer and
      information about the Artifact, enabling Transparency Service
      Registration Policies to validate the Signed Statement.  A Signed
      Statement is a COSE Envelope wrapped around a Statement, binding
      the metadata in the Envelope to the Statement.  In COSE, an
      Envelope consists of a protected header (included in the Issuer's
      signature) and an unprotected header (not included in the Issuer's
      signature).

   Equivocation:  a state where a Transparency Service provides
      inconsistent proofs to Relying Parties, containing conflicting
      claims about the Signed Statement bound at a given position in the
      Verifiable Data Structure.

   Issuer:  an identifier representing an organization, device, user, or
      entity securing Statements about supply chain Artifacts.  An
      Issuer may be the owner or author of Artifacts, or an independent
      third party such as an Auditor, reviewer or an endorser.  In SCITT
      Statements and Receipts, the iss Claim is a member of the COSE
      header parameter 15: CWT Claims defined in [RFC9597], which embeds
      a CWT Claim Set [RFC8392] within the protected header of a COSE
      Envelope.  This document uses the terms "Issuer", and "Subject" as
      described in [RFC8392], however the usage is consistent with the
      broader interpretation of these terms in both JOSE and COSE, and
      the guidance in [RFC8725] generally applies the COSE equivalent
      terms with consistent semantics.




Birkholz, et al.          Expires 13 April 2026                [Page 11]


Internet-Draft             SCITT Architecture               October 2025


   Non-equivocation:  a state where all proofs provided by the
      Transparency Service to Relying Parties are produced from a single
      Verifiable Data Structure describing a unique sequence of Signed
      Statements and are therefore consistent [EQUIVOCATION].  Over
      time, an Issuer may register new Signed Statements about an
      Artifact in a Transparency Service with new information.  However,
      the consistency of a collection of Signed Statements about the
      Artifact can be checked by all Relying Parties.

   Receipt:  a cryptographic proof that a Signed Statement is included
      in the Verifiable Data Structure.  See
      [I-D.draft-ietf-cose-merkle-tree-proofs] for implementations.
      Receipts are signed proofs of verifiable data-structure
      properties.  Receipt Profiles implemented by a Transparency
      Service MUST support inclusion proofs and MAY support other proof
      types, such as consistency proofs.

   Registration:  the process of submitting a Signed Statement to a
      Transparency Service, applying the Transparency Service's
      Registration Policy, adding to the Verifiable Data Structure, and
      producing a Receipt.

   Registration Policy:  the pre-condition enforced by the Transparency
      Service before registering a Signed Statement, based on
      information in the non-opaque header and metadata contained in its
      COSE Envelope.

   Relying Party:  Relying Parties consumes Transparent Statements,
      verifying their proofs and inspecting the Statement payload,
      either before using corresponding Artifacts, or later to audit an
      Artifact's provenance on the supply chain.

   Signed Statement:  an identifiable and non-repudiable Statement about
      an Artifact signed by an Issuer.  In SCITT, Signed Statements are
      encoded as COSE signed objects; the payload of the COSE structure
      contains the issued Statement.

   Attestation:  [NIST.SP.1800-19] defines "attestation" as "The process
      of providing a digital signature for a set of measurements
      securely stored in hardware, and then having the requester
      validate the signature and the set of measurements."  NIST
      guidance "Software Supply Chain Security Guidance EO 14028" uses
      the definition from [NIST_EO14028], which states that an
      "attestation" is "The issue of a statement, based on a decision,
      that fulfillment of specified requirements has been
      demonstrated.".  It is often useful for the intended audience to
      qualify the term "attestation" in their specific context to avoid
      confusion and ambiguity.



Birkholz, et al.          Expires 13 April 2026                [Page 12]


Internet-Draft             SCITT Architecture               October 2025


   Statement:  any serializable information about an Artifact.  To help
      interpretation of Statements, they must be tagged with a relevant
      media type (as specified in [RFC6838]).  A Statement may represent
      a Software Bill Of Materials (SBOM) that lists the ingredients of
      a software Artifact, an endorsement or attestation about an
      Artifact, indicate the End of Life (EOL), redirection to a newer
      version, or any content an Issuer wishes to publish about an
      Artifact.  Additional Statements about an Artifact are correlated
      by the Subject Claim as defined in the IANA CWT [IANA.cwt]
      registry and used as a protected header parameter as defined in
      [RFC9597].  The Statement is considered opaque to Transparency
      Service, and MAY be encrypted.

   Statement Sequence:  a sequence of Signed Statements captured by a
      Verifiable Data Structure.  See Verifiable Data Structure.

   Subject:  an identifier, defined by the Issuer, which represents the
      organization, device, user, entity, or Artifact about which
      Statements (and Receipts) are made and by which a logical
      collection of Statements can be grouped.  It is possible that
      there are multiple Statements about the same Artifact.  In these
      cases, distinct Issuers (iss) might agree to use the sub CWT
      Claim, defined in [RFC8392], to create a coherent sequence of
      Signed Statements about the same Artifact and Relying Parties can
      leverage sub to ensure completeness and Non-equivocation across
      Statements by identifying all Transparent Statements associated to
      a specific Subject.

   Transparency Service:  an entity that maintains and extends the
      Verifiable Data Structure and endorses its state.  The identity of
      a Transparency Service is captured by a public key that must be
      known by Relying Parties in order to validate Receipts.

   Transparent Statement:  a Signed Statement that is augmented with a
      Receipt created via Registration in a Transparency Service.  The
      Receipt is stored in the unprotected header of COSE Envelope of
      the Signed Statement.  A Transparent Statement remains a valid
      Signed Statement and may be registered again in a different
      Transparency Service.

   Verifiable Data Structure:  a data structure which supports one or
      more proof types, such as "inclusion proofs" or "consistency
      proofs", for Signed Statements as they are Registered to a
      Transparency Service.  SCITT supports multiple Verifiable Data
      Structures and Receipt formats as defined in
      [I-D.draft-ietf-cose-merkle-tree-proofs], accommodating different
      Transparency Service implementations.




Birkholz, et al.          Expires 13 April 2026                [Page 13]


Internet-Draft             SCITT Architecture               October 2025

""",
            "expected_clean_output": """3.  Terminology

   The terms defined in this section have special meaning in the context
   of Supply Chain Integrity, Transparency, and Trust, and are used
   throughout this document.

   This document has been developed in coordination with the COSE, OAUTH
   and RATS WG and uses terminology common to these working groups as
   much as possible.

   When used in text, the corresponding terms are capitalized.  To
   ensure readability, only a core set of terms is included in this
   section.

   The terms "header", "payload", and "to-be-signed bytes" are defined
   in [STD96].








   The term "claim" is defined in [RFC8392].

   Append-only Log:  a Statement Sequence comprising the entire
      registration history of the Transparency Service.  To make the
      Append-only property verifiable and transparent, the Transparency
      Service defines how Signed Statements are made available to
      Auditors.

   Artifact:  a physical or non-physical item that is moving along a
      supply chain.

   Auditor:  an entity that checks the correctness and consistency of
      all Transparent Statements, or the transparent Statement Sequence,
      issued by a Transparency Service.  An Auditor is an example of a
      specialized Relying Party.

   Client:  an application making protected Transparency Service
      resource requests on behalf of the resource owner and with its
      authorization.

   Envelope:  metadata, created by the Issuer to produce a Signed
      Statement.  The Envelope contains the identity of the Issuer and
      information about the Artifact, enabling Transparency Service
      Registration Policies to validate the Signed Statement.  A Signed
      Statement is a COSE Envelope wrapped around a Statement, binding
      the metadata in the Envelope to the Statement.  In COSE, an
      Envelope consists of a protected header (included in the Issuer's
      signature) and an unprotected header (not included in the Issuer's
      signature).

   Equivocation:  a state where a Transparency Service provides
      inconsistent proofs to Relying Parties, containing conflicting
      claims about the Signed Statement bound at a given position in the
      Verifiable Data Structure.

   Issuer:  an identifier representing an organization, device, user, or
      entity securing Statements about supply chain Artifacts.  An
      Issuer may be the owner or author of Artifacts, or an independent
      third party such as an Auditor, reviewer or an endorser.  In SCITT
      Statements and Receipts, the iss Claim is a member of the COSE
      header parameter 15: CWT Claims defined in [RFC9597], which embeds
      a CWT Claim Set [RFC8392] within the protected header of a COSE
      Envelope.  This document uses the terms "Issuer", and "Subject" as
      described in [RFC8392], however the usage is consistent with the
      broader interpretation of these terms in both JOSE and COSE, and
      the guidance in [RFC8725] generally applies the COSE equivalent
      terms with consistent semantics.








   Non-equivocation:  a state where all proofs provided by the
      Transparency Service to Relying Parties are produced from a single
      Verifiable Data Structure describing a unique sequence of Signed
      Statements and are therefore consistent [EQUIVOCATION].  Over
      time, an Issuer may register new Signed Statements about an
      Artifact in a Transparency Service with new information.  However,
      the consistency of a collection of Signed Statements about the
      Artifact can be checked by all Relying Parties.

   Receipt:  a cryptographic proof that a Signed Statement is included
      in the Verifiable Data Structure.  See
      [I-D.draft-ietf-cose-merkle-tree-proofs] for implementations.
      Receipts are signed proofs of verifiable data-structure
      properties.  Receipt Profiles implemented by a Transparency
      Service MUST support inclusion proofs and MAY support other proof
      types, such as consistency proofs.

   Registration:  the process of submitting a Signed Statement to a
      Transparency Service, applying the Transparency Service's
      Registration Policy, adding to the Verifiable Data Structure, and
      producing a Receipt.

   Registration Policy:  the pre-condition enforced by the Transparency
      Service before registering a Signed Statement, based on
      information in the non-opaque header and metadata contained in its
      COSE Envelope.

   Relying Party:  Relying Parties consumes Transparent Statements,
      verifying their proofs and inspecting the Statement payload,
      either before using corresponding Artifacts, or later to audit an
      Artifact's provenance on the supply chain.

   Signed Statement:  an identifiable and non-repudiable Statement about
      an Artifact signed by an Issuer.  In SCITT, Signed Statements are
      encoded as COSE signed objects; the payload of the COSE structure
      contains the issued Statement.

   Attestation:  [NIST.SP.1800-19] defines "attestation" as "The process
      of providing a digital signature for a set of measurements
      securely stored in hardware, and then having the requester
      validate the signature and the set of measurements."  NIST
      guidance "Software Supply Chain Security Guidance EO 14028" uses
      the definition from [NIST_EO14028], which states that an
      "attestation" is "The issue of a statement, based on a decision,
      that fulfillment of specified requirements has been
      demonstrated.".  It is often useful for the intended audience to
      qualify the term "attestation" in their specific context to avoid
      confusion and ambiguity.







   Statement:  any serializable information about an Artifact.  To help
      interpretation of Statements, they must be tagged with a relevant
      media type (as specified in [RFC6838]).  A Statement may represent
      a Software Bill Of Materials (SBOM) that lists the ingredients of
      a software Artifact, an endorsement or attestation about an
      Artifact, indicate the End of Life (EOL), redirection to a newer
      version, or any content an Issuer wishes to publish about an
      Artifact.  Additional Statements about an Artifact are correlated
      by the Subject Claim as defined in the IANA CWT [IANA.cwt]
      registry and used as a protected header parameter as defined in
      [RFC9597].  The Statement is considered opaque to Transparency
      Service, and MAY be encrypted.

   Statement Sequence:  a sequence of Signed Statements captured by a
      Verifiable Data Structure.  See Verifiable Data Structure.

   Subject:  an identifier, defined by the Issuer, which represents the
      organization, device, user, entity, or Artifact about which
      Statements (and Receipts) are made and by which a logical
      collection of Statements can be grouped.  It is possible that
      there are multiple Statements about the same Artifact.  In these
      cases, distinct Issuers (iss) might agree to use the sub CWT
      Claim, defined in [RFC8392], to create a coherent sequence of
      Signed Statements about the same Artifact and Relying Parties can
      leverage sub to ensure completeness and Non-equivocation across
      Statements by identifying all Transparent Statements associated to
      a specific Subject.

   Transparency Service:  an entity that maintains and extends the
      Verifiable Data Structure and endorses its state.  The identity of
      a Transparency Service is captured by a public key that must be
      known by Relying Parties in order to validate Receipts.

   Transparent Statement:  a Signed Statement that is augmented with a
      Receipt created via Registration in a Transparency Service.  The
      Receipt is stored in the unprotected header of COSE Envelope of
      the Signed Statement.  A Transparent Statement remains a valid
      Signed Statement and may be registered again in a different
      Transparency Service.

   Verifiable Data Structure:  a data structure which supports one or
      more proof types, such as "inclusion proofs" or "consistency
      proofs", for Signed Statements as they are Registered to a
      Transparency Service.  SCITT supports multiple Verifiable Data
      Structures and Receipt formats as defined in
      [I-D.draft-ietf-cose-merkle-tree-proofs], accommodating different
      Transparency Service implementations.







""",
        },
        {
            "query": "2.1",
            "expected_output": """2.1.  Generic SSC Problem Statement

   Supply chain security is a prerequisite to protecting consumers and
   minimizing economic, public health, and safety threats.  Supply chain
   security has historically focused on risk management practices to
   safeguard logistics, meet regulatory requirements, forecast demand,
   and optimize inventory.  While these elements are foundational to a
   healthy supply chain, an integrated cyber-security-based perspective
   of the software supply chains remains broadly undefined.  Recently,
   the global community has experienced numerous supply chain attacks
   targeting weaknesses in software supply chains.  As illustrated in
   Figure 1, a software supply chain attack may leverage one or more
   life-cycle stages and directly or indirectly target the component.





























Birkholz, et al.          Expires 13 April 2026                 [Page 5]


Internet-Draft             SCITT Architecture               October 2025


         Dependencies        Malicious 3rd-party package or version
              |
              |
        +-----+-----+
        |           |
        |   Code    |        Compromise source control
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Malicious plug-ins
        |  Commit   |        Malicious commit
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Modify build tasks or the build environment
        |   Build   |        Poison the build agent/compiler
        |           |        Tamper with build cache
        +-----+-----+
              |
        +-----+-----+
        |           |        Compromise test tools
        |    Test   |        Falsification of test results
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Use bad packages
        |  Package  |        Compromise package repository
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Modify release tasks
        |  Release  |        Modify build drop prior to release
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |
        |  Deploy   |        Tamper with versioning and update process
        |           |
        +-----------+

                  Figure 1: Example SSC Life-Cycle Threats





Birkholz, et al.          Expires 13 April 2026                 [Page 6]


Internet-Draft             SCITT Architecture               October 2025


   DevSecOps, as defined in [NIST.SP.800-204C], often depends on third-
   party and open-source software.  These dependencies can be quite
   complex throughout the supply chain, so checking provenance and
   traceability throughout their lifecycle is difficult.  There is a
   need for manageable auditability and accountability of digital
   products.  Typically, the range of types of statements about digital
   products (and their dependencies) is vast, heterogeneous, and can
   differ between community policy requirements.  Taking the type and
   structure of all statements about digital products into account might
   not be possible.  Examples of statements may include commit
   signatures, build environment and parameters, software bill of
   materials, static and dynamic application security testing results,
   fuzz testing results, release approvals, deployment records,
   vulnerability scan results, and patch logs.  In consequence, instead
   of trying to understand and describe the detailed syntax and
   semantics of every type of statement about digital products, the
   SCITT architecture focuses on ensuring statement authenticity,
   visibility/transparency, and intends to provide scalable
   accessibility.  Threats and practical issues can also arise from
   unintended side-effects of using security techniques outside their
   proper bounds.  For instance digital signatures may fail to verify
   past their expiry date even though the signed item itself remains
   completely valid.  Or a signature may verify even though the
   information it is securing is now found unreliable but fine-grained
   revocation is too hard.

   Lastly, where data exchange underpins serious business decision-
   making, it is important to hold the producers of those data to a
   higher standard of auditability.  The SCITT architecture provides
   mechanisms and structures for ensuring that the makers of
   authoritative statements can be held accountable and not hide or
   shred the evidence when it becomes inconvenient later.

   The following use cases illustrate the scope of SCITT and elaborate
   on the generic problem statement above.
""",
            "expected_clean_output": """2.1.  Generic SSC Problem Statement

   Supply chain security is a prerequisite to protecting consumers and
   minimizing economic, public health, and safety threats.  Supply chain
   security has historically focused on risk management practices to
   safeguard logistics, meet regulatory requirements, forecast demand,
   and optimize inventory.  While these elements are foundational to a
   healthy supply chain, an integrated cyber-security-based perspective
   of the software supply chains remains broadly undefined.  Recently,
   the global community has experienced numerous supply chain attacks
   targeting weaknesses in software supply chains.  As illustrated in
   Figure 1, a software supply chain attack may leverage one or more
   life-cycle stages and directly or indirectly target the component.

































         Dependencies        Malicious 3rd-party package or version
              |
              |
        +-----+-----+
        |           |
        |   Code    |        Compromise source control
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Malicious plug-ins
        |  Commit   |        Malicious commit
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Modify build tasks or the build environment
        |   Build   |        Poison the build agent/compiler
        |           |        Tamper with build cache
        +-----+-----+
              |
        +-----+-----+
        |           |        Compromise test tools
        |    Test   |        Falsification of test results
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Use bad packages
        |  Package  |        Compromise package repository
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |        Modify release tasks
        |  Release  |        Modify build drop prior to release
        |           |
        +-----+-----+
              |
        +-----+-----+
        |           |
        |  Deploy   |        Tamper with versioning and update process
        |           |
        +-----------+

                  Figure 1: Example SSC Life-Cycle Threats









   DevSecOps, as defined in [NIST.SP.800-204C], often depends on third-
   party and open-source software.  These dependencies can be quite
   complex throughout the supply chain, so checking provenance and
   traceability throughout their lifecycle is difficult.  There is a
   need for manageable auditability and accountability of digital
   products.  Typically, the range of types of statements about digital
   products (and their dependencies) is vast, heterogeneous, and can
   differ between community policy requirements.  Taking the type and
   structure of all statements about digital products into account might
   not be possible.  Examples of statements may include commit
   signatures, build environment and parameters, software bill of
   materials, static and dynamic application security testing results,
   fuzz testing results, release approvals, deployment records,
   vulnerability scan results, and patch logs.  In consequence, instead
   of trying to understand and describe the detailed syntax and
   semantics of every type of statement about digital products, the
   SCITT architecture focuses on ensuring statement authenticity,
   visibility/transparency, and intends to provide scalable
   accessibility.  Threats and practical issues can also arise from
   unintended side-effects of using security techniques outside their
   proper bounds.  For instance digital signatures may fail to verify
   past their expiry date even though the signed item itself remains
   completely valid.  Or a signature may verify even though the
   information it is securing is now found unreliable but fine-grained
   revocation is too hard.

   Lastly, where data exchange underpins serious business decision-
   making, it is important to hold the producers of those data to a
   higher standard of auditability.  The SCITT architecture provides
   mechanisms and structures for ensuring that the makers of
   authoritative statements can be held accountable and not hide or
   shred the evidence when it becomes inconvenient later.

   The following use cases illustrate the scope of SCITT and elaborate
   on the generic problem statement above.
""",
        },
    ],
}


class TestParseSections(unittest.TestCase):
    def test_skips_toc_dot_leader_entries(self):
        lines = [
            "",
            "1.  Intro .......... 3",
            "",
            "1.  Intro",
            "",
            "Body",
        ]
        sections = extract.parse_sections(lines)
        self.assertEqual(len(sections), 1)
        self.assertEqual(sections[0][0], "1")
        self.assertEqual(sections[0][1], "Intro")


class TestExtractFromFiles(unittest.TestCase):
    def test_extract_by_query(self):
        if not EXTRACT_CASES:
            self.skipTest("No extract cases configured")

        base_dir = Path(__file__).resolve().parents[1]
        for rel_path, cases in EXTRACT_CASES.items():
            path = base_dir / rel_path
            self.assertTrue(path.exists(), f"Missing test file: {rel_path}")
            text = path.read_text(encoding="utf-8", errors="replace")
            lines = text.splitlines()
            sections = extract.parse_sections(lines)

            for case in cases:
                query = case["query"]
                expected_output = case["expected_output"]
                with self.subTest(file=rel_path, query=query):
                    target = extract.find_section(sections, query)
                    self.assertIsNotNone(target, f"Missing section for query {query}")
                    chunk_lines = extract.extract(lines, sections, target)
                    output = "\n".join(chunk_lines)
                    self.assertEqual(output, expected_output)
                    if "expected_clean_output" in case:
                        clean_output = "\n".join(extract.clean_lines(chunk_lines))
                        self.assertEqual(clean_output, case["expected_clean_output"])


if __name__ == "__main__":
    unittest.main()
