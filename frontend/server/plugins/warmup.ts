import { getNatsConnection } from "~~/server/utils/nats";
import { getTemporalClient } from "~~/server/utils/temporal";

export default defineNitroPlugin(() => {
  // Dial NATS and Temporal at boot so the first workflow run doesn't
  // pay the handshake latency on top of Nitro's cold-compile delay.
  getNatsConnection().catch((error) => {
    console.warn("warmup: NATS dial failed (will retry on first subscribe)", error);
  });
  getTemporalClient().catch((error) => {
    console.warn("warmup: Temporal dial failed (will retry on first use)", error);
  });
});
