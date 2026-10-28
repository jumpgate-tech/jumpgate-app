import { expect, test } from "bun:test";
import { maskURL } from "../src/maskurl.ts";

test("maskURL hides userinfo, keys and query values", () => {
  expect(maskURL("http://127.0.0.1:8545")).toBe("http://127.0.0.1:8545");
  expect(maskURL("https://user:pw@rpc.example/x")).toBe("https://***/x");
  expect(maskURL("https://rpc.example/v3/0123456789abcdef0123")).toBe("https://rpc.example/v3/***");
  expect(maskURL("https://rpc.example/key/abc")).toBe("https://rpc.example/key/***");
  expect(maskURL("https://rpc.example/x?apikey=s3cret&chain=1")).toBe("https://rpc.example/x?apikey=***&chain=***");
  expect(maskURL("https://rpc.example/x#frag")).toBe("https://rpc.example/x#***");
});
