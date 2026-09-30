<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { EncryptionStartRequest } from "~~/shared/types";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
  scenario: EncryptionStartRequest["scenario"];
}>();

const lang = useCodeLang();

type StepKey = "encode" | "decode" | "register";

interface EncryptionSource extends CodeSource {
  stepLines: Record<StepKey, [number, number]>;
}

// All four snippets implement the same AES-256-GCM PayloadCodec and show how
// to register it on the client (and, in TypeScript, on the worker too, which
// does not inherit the client's data converter). Keep them structurally aligned
// — any change to one must land in the other three (project-memory:
// feedback_codeviewer_snippet_sync).
const SOURCES: Record<CodeLang, EncryptionSource> = {
  go: {
    label: "Go",
    lines: [
      "// Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "// Payload carrying metadata + nonce||ciphertext||authTag.",
      "func (c *EncryptionCodec) Encode(in []*commonpb.Payload) ([]*commonpb.Payload, error) {",
      "    gcm, _ := newGCM(c.Key) // errors elided for brevity",
      "    out := make([]*commonpb.Payload, len(in))",
      "    for i, p := range in {",
      "        plaintext, _ := proto.Marshal(p)",
      "        nonce := make([]byte, gcm.NonceSize())",
      "        _, _ = io.ReadFull(rand.Reader, nonce)",
      "        out[i] = &commonpb.Payload{",
      '            Metadata: map[string][]byte{"encoding": []byte("binary/encrypted")},',
      "            Data:     gcm.Seal(nonce, nonce, plaintext, nil), // appends to nonce",
      "        }",
      "    }",
      "    return out, nil",
      "}",
      "",
      "// Decode: reverse Encode — split nonce/ciphertext, open, unmarshal.",
      "func (c *EncryptionCodec) Decode(in []*commonpb.Payload) ([]*commonpb.Payload, error) {",
      "    gcm, _ := newGCM(c.Key)",
      "    out := make([]*commonpb.Payload, len(in))",
      "    for i, p := range in {",
      '        if string(p.Metadata["encoding"]) != "binary/encrypted" {',
      "            out[i] = p",
      "            continue",
      "        }",
      "        ns := gcm.NonceSize()",
      "        nonce, ct := p.Data[:ns], p.Data[ns:]",
      "        plaintext, err := gcm.Open(nil, nonce, ct, nil)",
      "        if err != nil { // wrong key or tampered ciphertext",
      '            return nil, fmt.Errorf("decrypt payload: %w", err)',
      "        }",
      "        var orig commonpb.Payload",
      "        _ = proto.Unmarshal(plaintext, &orig)",
      "        out[i] = &orig",
      "    }",
      "    return out, nil",
      "}",
      "",
      "// Register: attach the codec to the client — workers built on it inherit it.",
      "c, err := client.Dial(client.Options{",
      "    DataConverter: converter.NewCodecDataConverter(",
      "        converter.GetDefaultDataConverter(),",
      "        &EncryptionCodec{Key: demoKey},",
      "    ),",
      "})",
    ],
    stepLines: {
      encode: [0, 15],
      decode: [17, 37],
      register: [39, 45],
    },
  },
  java: {
    label: "Java",
    lines: [
      "// Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "// Payload carrying metadata + nonce||ciphertext||authTag.",
      "@Override",
      "public List<Payload> encode(List<Payload> in) {",
      "    var out = new ArrayList<Payload>(in.size());",
      "    try {",
      "        for (var p : in) {",
      "            byte[] plaintext = p.toByteArray();",
      "            byte[] nonce = new byte[12];",
      "            new SecureRandom().nextBytes(nonce);",
      '            var cipher = Cipher.getInstance("AES/GCM/NoPadding");',
      "            cipher.init(Cipher.ENCRYPT_MODE, keySpec, new GCMParameterSpec(128, nonce));",
      "            byte[] sealed = cipher.doFinal(plaintext);",
      "            out.add(Payload.newBuilder()",
      '                .putMetadata("encoding", ByteString.copyFromUtf8("binary/encrypted"))',
      "                .setData(ByteString.copyFrom(nonce).concat(ByteString.copyFrom(sealed)))",
      "                .build());",
      "        }",
      "    } catch (GeneralSecurityException e) {",
      "        throw new PayloadCodecException(e);",
      "    }",
      "    return out;",
      "}",
      "",
      "// Decode: reverse encode — split nonce/ciphertext, open, unmarshal.",
      "@Override",
      "public List<Payload> decode(List<Payload> in) {",
      "    var out = new ArrayList<Payload>(in.size());",
      "    try {",
      "        for (var p : in) {",
      '            var enc = p.getMetadataOrDefault("encoding", ByteString.EMPTY).toStringUtf8();',
      '            if (!"binary/encrypted".equals(enc)) { out.add(p); continue; }',
      "            byte[] raw = p.getData().toByteArray();",
      "            byte[] nonce = Arrays.copyOfRange(raw, 0, 12);",
      "            byte[] ct = Arrays.copyOfRange(raw, 12, raw.length);",
      '            var cipher = Cipher.getInstance("AES/GCM/NoPadding");',
      "            cipher.init(Cipher.DECRYPT_MODE, keySpec, new GCMParameterSpec(128, nonce));",
      "            out.add(Payload.parseFrom(cipher.doFinal(ct)));",
      "        }",
      "    } catch (GeneralSecurityException | InvalidProtocolBufferException e) {",
      "        // Wrong key, tampered ciphertext, or a corrupt inner Payload.",
      "        throw new PayloadCodecException(e);",
      "    }",
      "    return out;",
      "}",
      "",
      "// Register: attach the codec to the client — workers built on it inherit it.",
      "var opts = WorkflowClientOptions.newBuilder()",
      "    .setDataConverter(new CodecDataConverter(",
      "        DefaultDataConverter.newDefaultInstance(),",
      "        List.of(new EncryptionCodec(demoKey))))",
      "    .build();",
      "var client = WorkflowClient.newInstance(service, opts);",
    ],
    stepLines: {
      encode: [0, 22],
      decode: [24, 44],
      register: [46, 52],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "// Payload carrying metadata + nonce||ciphertext||authTag.",
      "async encode(payloads: Payload[]): Promise<Payload[]> {",
      "    return payloads.map((p) => {",
      "        const plaintext = temporal.api.common.v1.Payload.encode(p).finish();",
      "        const nonce = randomBytes(12);",
      '        const cipher = createCipheriv("aes-256-gcm", this.key, nonce);',
      "        const ct = Buffer.concat([cipher.update(plaintext), cipher.final()]);",
      "        const tag = cipher.getAuthTag();",
      "        return {",
      '            metadata: { encoding: Buffer.from("binary/encrypted") },',
      "            data: Buffer.concat([nonce, ct, tag]),",
      "        };",
      "    });",
      "}",
      "",
      "// Decode: reverse encode — split nonce/ciphertext/tag, open, unmarshal.",
      "async decode(payloads: Payload[]): Promise<Payload[]> {",
      "    return payloads.map((p) => {",
      '        const enc = Buffer.from(p.metadata?.encoding ?? []).toString("utf8");',
      '        if (enc !== "binary/encrypted") return p;',
      "        const raw = Buffer.from(p.data ?? []);",
      "        const nonce = raw.subarray(0, 12);",
      "        const tag = raw.subarray(raw.length - 16);",
      "        const ct = raw.subarray(12, raw.length - 16);",
      '        const decipher = createDecipheriv("aes-256-gcm", this.key, nonce);',
      "        decipher.setAuthTag(tag);",
      "        const plaintext = Buffer.concat([decipher.update(ct), decipher.final()]);",
      "        return temporal.api.common.v1.Payload.decode(plaintext);",
      "    });",
      "}",
      "",
      "// Register: a TS Worker does not inherit the client's data converter,",
      "// so both get the codec.",
      "const dataConverter = { payloadCodecs: [new EncryptionCodec(demoKey)] };",
      "const client = new Client({ connection, dataConverter });",
      "const worker = await Worker.create({",
      '    taskQueue: "patterns-encryption-encrypted",',
      '    workflowsPath: require.resolve("./workflows"),',
      "    activities,",
      "    dataConverter,",
      "});",
    ],
    stepLines: {
      encode: [0, 14],
      decode: [16, 30],
      register: [32, 41],
    },
  },
  python: {
    label: "Python",
    lines: [
      "class EncryptionCodec(PayloadCodec):",
      "    # Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "    # Payload carrying metadata + nonce||ciphertext||authTag.",
      "    async def encode(self, payloads: Sequence[Payload]) -> list[Payload]:",
      "        out: list[Payload] = []",
      "        for p in payloads:",
      "            plaintext = p.SerializeToString()",
      "            nonce = secrets.token_bytes(12)",
      "            sealed = AESGCM(self.key).encrypt(nonce, plaintext, None)",
      "            out.append(Payload(",
      '                metadata={"encoding": b"binary/encrypted"},',
      "                data=nonce + sealed,",
      "            ))",
      "        return out",
      "",
      "    # Decode: reverse encode — split nonce/ciphertext, open, unmarshal.",
      "    async def decode(self, payloads: Sequence[Payload]) -> list[Payload]:",
      "        out: list[Payload] = []",
      "        for p in payloads:",
      '            if p.metadata.get("encoding", b"") != b"binary/encrypted":',
      "                out.append(p)",
      "                continue",
      "            nonce, ct = p.data[:12], p.data[12:]",
      "            plaintext = AESGCM(self.key).decrypt(nonce, ct, None)",
      "            orig = Payload()",
      "            orig.ParseFromString(plaintext)",
      "            out.append(orig)",
      "        return out",
      "",
      "# Register: attach the codec to the client — workers built on it inherit it.",
      "client = await Client.connect(",
      '    "localhost:7233",',
      "    data_converter=dataclasses.replace(",
      "        DataConverter.default, payload_codec=EncryptionCodec(demo_key)),",
      ")",
    ],
    stepLines: {
      encode: [1, 13],
      decode: [15, 27],
      register: [29, 34],
    },
  },
};

const currentHighlight = computed<[number, number] | null>(() => {
  // The clear scenario runs without the codec, so none of this code executes.
  if (props.scenario === "clear") return null;

  // Terminal events are skipped on purpose, so after completion the last
  // encode (the workflow result) stays highlighted.
  const src = SOURCES[lang.value];
  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    // Every activity boundary crosses the codec twice: the worker decodes the
    // activity input before it runs, then encodes its result once it returns.
    if (env.type === "progress.step.started") return src.stepLines.decode;
    if (env.type === "progress.step.completed") return src.stepLines.encode;
  }
  return src.stepLines.register;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
