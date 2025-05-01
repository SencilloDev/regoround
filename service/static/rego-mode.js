const builtins = [
  "abs",
  "all",
  "and",
  "any",
  "array.concat",
  "array.reverse",
  "array.slice",
  "assign",
  "base64.decode",
  "base64.encode",
  "base64.is_valid",
  "base64url.decode",
  "base64url.encode",
  "base64url.encode_no_pad",
  "bits.and",
  "bits.lsh",
  "bits.negate",
  "bits.or",
  "bits.rsh",
  "bits.xor",
  "cast_array",
  "cast_boolean",
  "cast_null",
  "cast_object",
  "cast_set",
  "cast_string",
  "ceil",
  "concat",
  "contains",
  "count",
  "crypto.hmac.equal",
  "crypto.hmac.md5",
  "crypto.hmac.sha1",
  "crypto.hmac.sha256",
  "crypto.hmac.sha512",
  "crypto.md5",
  "crypto.parse_private_keys",
  "crypto.sha1",
  "crypto.sha256",
  "crypto.x509.parse_and_verify_certificates",
  "crypto.x509.parse_and_verify_certificates_with_options",
  "crypto.x509.parse_certificate_request",
  "crypto.x509.parse_certificates",
  "crypto.x509.parse_keypair",
  "crypto.x509.parse_rsa_private_key",
  "div",
  "endswith",
  "eq",
  "equal",
  "floor",
  "format_int",
  "glob.match",
  "glob.quote_meta",
  "graph.reachable",
  "graph.reachable_paths",
  "graphql.is_valid",
  "graphql.parse",
  "graphql.parse_and_verify",
  "graphql.parse_query",
  "graphql.parse_schema",
  "graphql.schema_is_valid",
  "gt",
  "gte",
  "hex.decode",
  "hex.encode",
  "http.send",
  "indexof",
  "indexof_n",
  "internal.member_2",
  "internal.member_3",
  "internal.print",
  "internal.test_case",
  "intersection",
  "io.jwt.decode",
  "io.jwt.decode_verify",
  "io.jwt.encode_sign",
  "io.jwt.encode_sign_raw",
  "io.jwt.verify_es256",
  "io.jwt.verify_es384",
  "io.jwt.verify_es512",
  "io.jwt.verify_hs256",
  "io.jwt.verify_hs384",
  "io.jwt.verify_hs512",
  "io.jwt.verify_ps256",
  "io.jwt.verify_ps384",
  "io.jwt.verify_ps512",
  "io.jwt.verify_rs256",
  "io.jwt.verify_rs384",
  "io.jwt.verify_rs512",
  "is_array",
  "is_boolean",
  "is_null",
  "is_number",
  "is_object",
  "is_set",
  "is_string",
  "json.filter",
  "json.is_valid",
  "json.marshal",
  "json.marshal_with_options",
  "json.match_schema",
  "json.patch",
  "json.remove",
  "json.unmarshal",
  "json.verify_schema",
  "lower",
  "lt",
  "lte",
  "max",
  "min",
  "minus",
  "mul",
  "neq",
  "net.cidr_contains",
  "net.cidr_contains_matches",
  "net.cidr_expand",
  "net.cidr_intersects",
  "net.cidr_is_valid",
  "net.cidr_merge",
  "net.cidr_overlap",
  "net.lookup_ip_addr",
  "numbers.range",
  "numbers.range_step",
  "object.filter",
  "object.get",
  "object.keys",
  "object.remove",
  "object.subset",
  "object.union",
  "object.union_n",
  "opa.runtime",
  "or",
  "plus",
  "print",
  "product",
  "providers.aws.sign_req",
  "rand.intn",
  "re_match",
  "regex.find_all_string_submatch_n",
  "regex.find_n",
  "regex.globs_match",
  "regex.is_valid",
  "regex.match",
  "regex.replace",
  "regex.split",
  "regex.template_match",
  "rego.metadata.chain",
  "rego.metadata.rule",
  "rego.parse_module",
  "rem",
  "replace",
  "round",
  "semver.compare",
  "semver.is_valid",
  "set_diff",
  "sort",
  "split",
  "sprintf",
  "startswith",
  "strings.any_prefix_match",
  "strings.any_suffix_match",
  "strings.count",
  "strings.render_template",
  "strings.replace_n",
  "strings.reverse",
  "substring",
  "sum",
  "time.add_date",
  "time.clock",
  "time.date",
  "time.diff",
  "time.format",
  "time.now_ns",
  "time.parse_duration_ns",
  "time.parse_ns",
  "time.parse_rfc3339_ns",
  "time.weekday",
  "to_number",
  "trace",
  "trim",
  "trim_left",
  "trim_prefix",
  "trim_right",
  "trim_space",
  "trim_suffix",
  "type_name",
  "union",
  "units.parse",
  "units.parse_bytes",
  "upper",
  "urlquery.decode",
  "urlquery.decode_object",
  "urlquery.encode",
  "urlquery.encode_object",
  "uuid.parse",
  "uuid.rfc4122",
  "walk",
  "yaml.is_valid",
  "yaml.marshal",
  "yaml.unmarshal",
];
CodeMirror.defineMode("rego", (editorOptions, modeOptions) => {
  const BUILTIN_REFERENCES_RE = new RegExp(
    "\\b(?:" + builtins.filter((c) => c.includes(".")).join("|") + ")\\b",
  );
  const BUILTINS_RE = new RegExp(
    "\\b(?:" + builtins.filter((c) => !c.includes(".")).join("|") + ")\\b",
  );
  const IDENTIFIER_RE = /^[A-Za-z_][A-Za-z_0-9]*/;
  const KEYWORDS_RE =
    /\b(?:as|default|else|import|not|with|some|in|every|if|contains)\b/;
  const NUMBER_RE =
    /^-?(?:(?:(?:0(?!\d+)|[1-9][0-9]*)(?:\.[0-9]+)?)|(?:\.[0-9]+))(?:[eE][-+]?[0-9]+)?/;
  const OPERATOR_RE = /^(?:&|;|\*|\+|-|\/|%|=|:=|==|!=|<|>|>=|<=|\|)/;
  const SCALAR_RE = /\b(?:true|false|null)\b/;

  const ast = modeOptions.ast;
  const packagesByName = {};
  const rulesByName = {};

  if (ast) {
    packagesByName[ast.package.getName()] = true;
    ast.imports.forEach((x) => (packagesByName[x.getName()] = true));
    ast.rules.forEach((x) => (rulesByName[x.getName()] = true));
  }

  function eatBuiltinReference(stream) {
    const match = stream.match(BUILTIN_REFERENCES_RE, false);
    if (match) {
      var n = match[0].lastIndexOf(".") + 1;
      var i = 0;

      stream.eatWhile(() => {
        return ++i < n;
      });

      return match[0].split(".").pop();
    }
  }

  function eatPathIdentifier(stream) {
    const identifier = eatIdentifier(stream);

    if (stream.eat(".")) {
      return;
    }

    return identifier;
  }

  function eatIdentifier(stream) {
    var match = stream.match(IDENTIFIER_RE);
    if (match) {
      return match[0];
    }
  }

  function eatString(stream) {
    let escaped = false;

    stream.eatWhile((c) => {
      if (escaped) {
        escaped = false;
      } else {
        if (c === '"') {
          return false;
        } else if (c === "\\") {
          escaped = true;
        }
      }

      return true;
    });

    stream.eat('"');
  }

  function inBacktickString(stream, state) {
    var next;
    while ((next = stream.next())) {
      // eslint-disable-line no-cond-assign
      if (next === "`") {
        state.tokenize = base;
        break;
      }
    }
    return "string-2";
  }

  function base(stream, state) {
    if (stream.eatSpace()) {
      return;
    } else if (state.builtin) {
      stream.match(state.builtin);
      state.builtin = "";
      return "builtin";
    } else if (state.inPackage) {
      if (stream.match(/\bas\b/)) {
        return "keyword";
      }
      const identifier = eatPathIdentifier(stream);
      if (identifier) {
        state.inPackage = false;
        return "variable-2";
      }
    } else if (state.inPath) {
      const identifier = eatPathIdentifier(stream);
      if (identifier) {
        state.inPath = false;
      }
    } else if (stream.eat('"')) {
      eatString(stream, state);
      return "string";
    } else if (stream.eat("`")) {
      return (state.tokenize = inBacktickString)(stream, state);
    } else if (stream.eat("#")) {
      stream.skipToEnd();
      return "comment";
    } else if (stream.match(/\bpackage\b/) || stream.match(/\bimport\b/)) {
      state.inPackage = true;
      return "keyword";
    } else if (stream.match(KEYWORDS_RE)) {
      return "keyword";
    } else if (stream.match(SCALAR_RE)) {
      return "atom";
    } else if (stream.match(/\binput\b/)) {
      return "variable-2";
    } else if (stream.match(NUMBER_RE)) {
      return "number";
    } else if (stream.eat(".")) {
      state.inPath = true;
    } else {
      const builtin = eatBuiltinReference(stream);
      if (builtin) {
        state.builtin = builtin;
      } else {
        const identifier = eatIdentifier(stream);
        if (identifier) {
          if (identifier === "_") {
            return "operator";
          } else if (BUILTINS_RE.test(identifier)) {
            return "builtin";
          } else if (rulesByName[identifier]) {
            return "def";
          } else if (packagesByName[identifier]) {
            return "variable-2";
          }
          return "variable";
        } else if (stream.match(OPERATOR_RE)) {
          return "operator";
        }
      }

      stream.next();
    }

    if (stream.pos === stream.start) {
      // We weren’t able to tokenize anything and `stream` is in exactly the
      // same place as it was at the beginning of the `token()` call. One way
      // that this can happen is if the stream is syntactically invalid (e.g.,
      // `bar.[x]`). Whatever the cause, we need to skip past the problematic
      // character.
      stream.next();
    }
  }

  return {
    token: function (stream, state) {
      return state.tokenize(stream, state);
    },

    startState: function () {
      return {
        builtin: "",
        inPackage: false,
        inPath: false,
        tokenize: base,
      };
    },

    lineComment: "#",
  };
});

CodeMirror.defineMIME("application/rego", "rego");
