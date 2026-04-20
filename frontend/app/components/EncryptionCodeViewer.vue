<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

type StepKey = "encode" | "decode" | "register";

interface EncryptionSource extends CodeSource {
  stepLines: Record<StepKey, [number, number]>;
}

// All four snippets implement the same AES-256-GCM PayloadCodec and show how
// the client wires it into its data converter. Keep them structurally aligned
// — any change to one must land in the other three (project-memory:
// feedback_codeviewer_snippet_sync).
const SOURCES: Record<CodeLang, EncryptionSource> = {
  go: {
    label: "Go",
    lines: [
      "// Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "// Payload carrying metadata + nonce||ciphertext||authTag.",
      "func (c *EncryptionCodec) Encode(in []*commonpb.Payload) ([]*commonpb.Payload, error) {",
      "    gcm, _ := newGCM(c.Key)",
      "    out := make([]*commonpb.Payload, len(in))",
      "    for i, p := range in {",
      "        plaintext, _ := proto.Marshal(p)",
      "        nonce := make([]byte, gcm.NonceSize())",
      "        _, _ = io.ReadFull(rand.Reader, nonce)",
      "        sealed := gcm.Seal(nil, nonce, plaintext, nil)",
      "        out[i] = &commonpb.Payload{",
      '            Metadata: map[string][]byte{"encoding": []byte("binary/encrypted")},',
      "            Data:     append(nonce, sealed...),",
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
      "        plaintext, _ := gcm.Open(nil, nonce, ct, nil)",
      "        var orig commonpb.Payload",
      "        _ = proto.Unmarshal(plaintext, &orig)",
      "        out[i] = &orig",
      "    }",
      "    return out, nil",
      "}",
      "",
      "// Register: attach the codec to the client's DataConverter.",
      "c, err := client.Dial(client.Options{",
      "    DataConverter: converter.NewCodecDataConverter(",
      "        converter.GetDefaultDataConverter(),",
      "        &EncryptionCodec{Key: demoKey},",
      "    ),",
      "})",
    ],
    stepLines: {
      encode: [0, 16],
      decode: [18, 35],
      register: [37, 43],
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
      "    for (var p : in) {",
      "        byte[] plaintext = p.toByteArray();",
      "        byte[] nonce = new byte[12];",
      "        new SecureRandom().nextBytes(nonce);",
      '        var cipher = Cipher.getInstance("AES/GCM/NoPadding");',
      "        cipher.init(Cipher.ENCRYPT_MODE, keySpec, new GCMParameterSpec(128, nonce));",
      "        byte[] sealed = cipher.doFinal(plaintext);",
      "        out.add(Payload.newBuilder()",
      '            .putMetadata("encoding", ByteString.copyFromUtf8("binary/encrypted"))',
      "            .setData(ByteString.copyFrom(concat(nonce, sealed)))",
      "            .build());",
      "    }",
      "    return out;",
      "}",
      "",
      "// Decode: reverse encode — split nonce/ciphertext, open, unmarshal.",
      "@Override",
      "public List<Payload> decode(List<Payload> in) {",
      "    var out = new ArrayList<Payload>(in.size());",
      "    for (var p : in) {",
      '        var enc = p.getMetadataOrDefault("encoding", ByteString.EMPTY).toStringUtf8();',
      '        if (!"binary/encrypted".equals(enc)) { out.add(p); continue; }',
      "        byte[] raw = p.getData().toByteArray();",
      "        byte[] nonce = Arrays.copyOfRange(raw, 0, 12);",
      "        byte[] ct    = Arrays.copyOfRange(raw, 12, raw.length);",
      '        var cipher = Cipher.getInstance("AES/GCM/NoPadding");',
      "        cipher.init(Cipher.DECRYPT_MODE, keySpec, new GCMParameterSpec(128, nonce));",
      "        out.add(Payload.parseFrom(cipher.doFinal(ct)));",
      "    }",
      "    return out;",
      "}",
      "",
      "// Register: attach the codec to the client's DataConverter.",
      "var opts = WorkflowClientOptions.newBuilder()",
      "    .setDataConverter(new CodecDataConverter(",
      "        DefaultDataConverter.newDefaultInstance(),",
      "        List.of(new EncryptionCodec(demoKey))))",
      "    .build();",
      "var client = WorkflowClient.newInstance(service, opts);",
    ],
    stepLines: {
      encode: [0, 18],
      decode: [20, 36],
      register: [38, 43],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      "// Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "// Payload carrying metadata + nonce||ciphertext||authTag.",
      "async encode(payloads: Payload[]): Promise<Payload[]> {",
      "    return payloads.map((p) => {",
      "        const plaintext = encodePayloadProto(p);",
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
      "        return decodePayloadProto(plaintext);",
      "    });",
      "}",
      "",
      "// Register: attach the codec to the client's DataConverter.",
      "const client = new Client({",
      "    connection,",
      '    namespace: "default",',
      "    dataConverter: { payloadCodecs: [new EncryptionCodec(demoKey)] },",
      "});",
    ],
    stepLines: {
      encode: [0, 13],
      decode: [15, 29],
      register: [31, 36],
    },
  },
  python: {
    label: "Python",
    lines: [
      "# Encode: marshal the Payload, seal with AES-256-GCM, return a new",
      "# Payload carrying metadata + nonce||ciphertext||authTag.",
      "class EncryptionCodec(PayloadCodec):",
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
      "# Register: attach the codec to the client's DataConverter.",
      "client = await Client.connect(",
      '    "localhost:7233",',
      "    data_converter=dataclasses.replace(",
      "        DataConverter.default, payload_codec=EncryptionCodec(demo_key)),",
      ")",
    ],
    stepLines: {
      encode: [0, 13],
      decode: [15, 27],
      register: [29, 34],
    },
  },
};

const TERMINAL_TYPES: ReadonlySet<string> = new Set([
  "progress.workflow.completed",
  "progress.workflow.failed",
  "encryption.order.completed",
]);

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  let lastStartedStep: string | null = null;

  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    if (TERMINAL_TYPES.has(env.type)) return null;
    if (env.type === "progress.step.started") {
      const step = (env.data as Record<string, unknown>).step;
      lastStartedStep = typeof step === "string" ? step : null;
      break;
    }
  }

  // Codec runs at every activity boundary: validate/charge exercise encode
  // (client → server); ship/receipt exercise decode (server → worker).
  if (lastStartedStep === "validate-order" || lastStartedStep === "charge-card")
    return src.stepLines.encode;
  if (lastStartedStep === "ship-order" || lastStartedStep === "send-receipt")
    return src.stepLines.decode;
  return src.stepLines.register;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
