# How I Write: Grammar, Mechanics, and Voice

Author: Layth M Qassem  
Date: 2025-08-22  
Purpose: Document my actual writing patterns so others can replicate them

I analyzed my SmartSig documentation, Confluence pages, and technical specs to extract how I actually write. Not theory. Patterns from real documents I've shipped.

## Sentence Construction

Complex sentences are fine when they carry substance. My SmartSig doc has sentences like "The system checks if this exact signature exists in our whitelist database, evaluates the text complexity when there's no match, then routes to either the regex-based pattern matching engine for simple medical abbreviations or the sequence-to-sequence LSTM model for natural language prescriptions." That's 44 words but every clause adds necessary information. The key is density of meaning, not arbitrary length limits.

Subject-verb-object, then context. "SmartSig processes prescriptions" then add conditions. Not "When prescriptions arrive from various sources, SmartSig processes them."

Active voice unless passive adds clarity. "We deployed v1.8.2" not "v1.8.2 was deployed." But "The prescription was validated" when the validator doesn't matter.

Parallel structure in lists. Lists always follow paragraphs that set them up. Never float bullets without context. My actual pattern from production docs explains what's coming, then lists:
- Checks if exact signature exists in whitelist
- Evaluates text complexity  
- Routes to appropriate engine

All start with verbs. Same grammatical shape. The paragraph before explained why we need these three steps.

## Punctuation That Works

Periods. My default. Short sentences. Clear stops. Look at my actual email: "Hi! Could you check if port 5432 is accessible from your end?" Two sentences, not one with a comma.

Commas for clarity, not decoration. "When confidence falls below the configured threshold (typically between 4.0 and 6.5 depending on the client), the field returns as null." That's a natural pause point, not arbitrary.

Parentheses for optional depth. I use these constantly: "(FDB) and (Medi-Span)" for vendor names, "(typically between 4.0 and 6.5)" for ranges. Readers can skip or read based on need.

Colons before lists, code, or definitions. "SmartSig Response:" then the JSON. "Step 1:" then the action. Never floating colons.

Em-dash sparingly. Three in 11 pages of SmartSig docs. That's my ratio. One sharp aside per topic max.

Hyphens in compound modifiers. "Real-time benefit tool" not "real time benefit tool." But "the tool works in real time" - no hyphen after the noun.

## Word Choice From My Actual Docs

Plain verbs I use constantly:
- use, build, check, return, process, validate, extract, map, route, fail

Concrete nouns from my work:
- field, table, endpoint, signature, prescription, dose, route, frequency, confidence

Numbers over adjectives. Not "fast processing" but "P50 latency: 25ms." Not "high accuracy" but "99.51% match rate."

Technical terms with immediate definition. "GCN_SEQNO (FDB) and (Medi-Span) are both generic-formulation identifiers." Term, vendor, then what it means.

## Tense Consistency

Present for system behavior: "SmartSig processes prescriptions"  
Past for what happened: "We deployed the fix at 14:32 UTC"  
Future only with dates: "We will migrate on 2025-09-15"  

Never vague future like "we plan to optimize." Either commit with a date or state the condition: "After validation passes, we migrate."

## Paragraph and Bullet Philosophy

Merge related thoughts into substantial paragraphs. Think of it like a transcript - if one person is talking about the same topic, keep it together. My SmartSig documentation doesn't scatter single sentences everywhere. When I explain authentication, it's one cohesive paragraph: "Authentication happens at multiple layers. The API gateway enforces client authentication and rate limiting on inbound requests. When SmartSig needs to call the model service, it uses HMAC-SHA256 signatures to authenticate those outbound calls. Each client gets a unique key pair - a public KEY_ID and a secret KEY_SECRET." That's four related sentences that belong together, not four separate lines.

Three solid paragraphs beat nine scattered lines. Dense information transfer wins over visual white space. My production docs have paragraphs running 4-6 sentences when explaining a complete concept. The whitelist explanation runs 85 words in one block because it's one complete thought about how the whitelist evolved from performance optimization to safety component. Breaking it into bullet points would destroy the narrative flow.

Bullets clarify concepts, never stand alone. Every bullet list in my docs follows a setup paragraph and serves a specific purpose:
- Breaking down a complex process into steps
- Listing specific examples of a general concept
- Showing alternative options or configurations
- Providing multiple code samples or test cases

Notice how that list followed a complete sentence explaining what's coming? That's the pattern. The paragraph provides context, the bullets provide specifics, then we return to paragraph form. Never just dump bullets without explaining why they're there.

Wrong way (scattered bullets):

Benefits:
- Fast
- Reliable  
- Scalable

Right way (integrated with context):

The new architecture delivers three measurable improvements over our legacy system. First, we reduced P50 latency from 65ms to 25ms by implementing connection pooling and query optimization. Second, we achieved 99.99% uptime over the last quarter through redundant failover mechanisms. Third, we can now handle 5000 concurrent connections, up from 500, through horizontal scaling across our Kubernetes cluster. These improvements came from:
- Implementing PgBouncer for connection pooling (40% latency reduction)
- Adding read replicas with automatic failover (eliminated single point of failure)
- Containerizing services for elastic scaling (10x capacity increase)

See how the bullets support the paragraph rather than replacing it? That's the pattern.

## How I Handle Technical Content

Code blocks with language tags:

```python
signature_hash = hmac.new(KEY_SECRET.encode(), request_body.encode(), hashlib.sha256).digest()
```

Inline code for field names: The `ORDER_MED_ID` field links to `PAT_ID`.

JSON/XML with real data:

```json
{
  "dose": {"value": "1", "score": "1.0"},
  "doseUnit": {"value": "287~~mg", "score": "1.0"}
}
```

SQL with comments only when needed:

```sql
SELECT om.ORDER_MED_ID, om.PAT_ID  
FROM ORDER_MED AS om  
JOIN CLARITY_MEDICATION AS med  -- links to medication master
```

## Platform-Specific Adjustments

Confluence: Full technical depth. Tables for mappings. Bold headers for scanning.

Jira: "Hi! Could you check if port 5432 is accessible?" Direct ask, version numbers, what I tried.

Slack: "textbookGPT would love to try it on pharmacotherapy textbooks!" Enthusiasm allowed. Exclamation points OK.

Email: "Quick update on SmartSig performance - we're hitting 99.51% accuracy. Details in Confluence: [link]"

## Grammar Rules I Actually Follow

Contractions in informal contexts. "It's" in Slack, "It is" in specs.

Oxford comma always. "Dose, route, and frequency" not "dose, route and frequency."

Which vs that. "The system, which runs on AWS, processes prescriptions" (non-restrictive). "The system that processes prescriptions runs on AWS" (restrictive).

Comprise correctly. "The system comprises three modules" not "is comprised of."

Data as singular when referring to a set. "This data shows" not "these data show" unless in pure research context.

## The Death List: Words and Phrases to Kill

Hard-Ban Words (never use)
- leverage — say "use"
- utilize — say "use"  
- synergy/synergize — delete entirely
- holistic — be specific instead
- innovative/innovation — show what's new with specifics
- disruptive — unless discussing actual market disruption with data
- cutting-edge/state-of-the-art — give version numbers instead
- world-class — compared to what?
- best-in-class — show the benchmark
- game-changing — cite the before/after metrics
- revolutionary — reserved for actual revolutions
- paradigm/paradigm shift — pretentious, delete
- thought leader/thought leadership — ego phrase, delete
- robust — show error rates instead
- seamless — nothing is seamless, show the integration points
- frictionless — there's always friction, be honest
- intuitive — show usability test results
- user-friendly — meaningless, show task completion rates
- empower/empowerment — corporate speak, say "enables"
- unlock — not a treasure chest, say "enable" or "allow"
- elevate — say "improve" with metrics
- deep dive — say "detailed analysis" or just "analysis"
- circle back — say "revisit" or "discuss again"
- low-hanging fruit — say "quick wins" or list specific items
- move the needle — show the actual metric movement
- boil the ocean — say "overly ambitious" 
- bandwidth (for human capacity) — say "time" or "capacity"

Soft-Ban Words (only with immediate numbers)
- scalable — must follow with "handles X requests/second"
- optimize — must show "reduced from X to Y"
- significant — must include p-value or percentage
- comprehensive — must list what's included
- efficient — must show time/resource savings
- reliable — must show uptime or error rate

Phrases That Make Me Delete Your Paragraph
- "In today's fast-paced world..."
- "In the rapidly evolving landscape of..."
- "It's not just X, it's Y"
- "This document aims to..."
- "This document seeks to..."
- "At the end of the day..."
- "For all intents and purposes..."
- "Needless to say..."
- "It goes without saying..."
- "It should be noted that..."
- "To clarify..." (just clarify)
- "To summarize..." (just summarize)
- "In conclusion..." (just conclude)
- "As you may already know..."
- "We're excited to announce..."
- "We're thrilled to share..."
- "Drive impact at scale"
- "Leverage synergies"
- "Best practices" (without listing them)
- "Industry standard" (without naming the standard)
- "Next generation" (unless comparing to specific previous generation)
- "Enterprise-grade" (meaningless)
- "Military-grade" (especially meaningless)

Empty Intensifiers (delete on sight)
- very, really, quite, fairly, somewhat, rather, pretty (as intensifier)
- actually (unless contrasting)
- literally (unless preventing misinterpretation)
- basically, essentially, fundamentally (unless teaching)
- simply (unless describing simple process)
- just (when minimizing)
- definitely, certainly, surely (unless contrasting uncertainty)

AI Pattern Phrases
- Moreover, Furthermore, Additionally, at sentence start
- It's worth noting that...
- Dive deeper into...
- Unpack this...
- Let's explore...
- Journey through...
- Triple anything: "fast, reliable, and efficient"
- Not only... but also...
- Both... and... (when simple "and" works)

## Emoji Rules

Never in documentation. Zero. None.

Slack status only:
- ✅ for completed
- ⚠️ for warning/risk  
- ⛔ for blocked
- ℹ️ for FYI

Banned everywhere:
🚀 (not "launching"), 💯 (not "100%"), 🔥 (not "on fire"), 
🎉 (not "celebrating"), 🙌 (not "praising"), 😍 (not "loving it"),
🤯 (not "mind-blown"), 💪 (not "strong"), 🏆 (not "winning")

## Spacing and Visual Flow

Merge over scatter. Related content stays together. If you're explaining how authentication works, that's one section with 2-3 solid paragraphs, not 8 single-line statements with blank lines between them. White space is for major topic transitions, not every thought.

Linter spacing = continuous flow. When the linter wants to break everything into tiny chunks, ignore it. My actual SmartSig doc has sections where 5-6 paragraphs flow continuously because they're building one complete explanation. The QA process description runs 250+ words without a break because it's one cohesive workflow.

Section breaks only for topic changes. Use spacing when you're actually changing subjects - from authentication to performance, from implementation to monitoring. Not between every paragraph that happens to discuss a slightly different aspect of the same thing.

## Formatting Crimes

Never do these:
- Center body text
- Use decorative fonts
- Add color for emphasis (use bold)
- Create ASCII art diagrams (use tables)
- Screenshot text (paste it)
- Use blockquotes for layout
- Add "Pro tip" callout boxes
- Create headers past H3
- Use horizontal rules as decoration
- Break every paragraph with blank lines (unless changing topics)
- Create single-sentence paragraphs throughout a document
- Use bullets without setup paragraphs

## Quick Grammar Fixes

Less vs fewer: Less time, fewer errors  
While vs although: While = during time, although = contrast  
Since vs because: Since = time unless obvious  
Affect vs effect: Affect = verb, effect = noun (usually)  
Ensure vs insure: Ensure = make certain, insure = insurance  
Compliment vs complement: Compliment = praise, complement = complete  
Comprise: The whole comprises the parts (never "comprised of")

## My Actual Templates

Decision:
We chose PostgreSQL over MySQL because connection pooling handles our 5000 concurrent users better. Benchmark: 40ms query time vs 65ms under load. Owner: Layth. Date: 2025-08-22.

Problem statement:
Prescriptions fail parsing when PRN appears without indication (n=147/10000 last week). Current regex expects "prn [condition]" pattern.

Instruction:
Run `make migrate DB_SCHEMA=2025_08_22`. Expect "Applied 3 migrations (0 warnings)." If you see version mismatch, check your .env points to prod schema.

Update:
SmartSig accuracy improved from 99.40% to 99.51% after whitelist expansion. Biggest gain: pediatric liquid formulations (+0.08pp). Next: IV compatibility rules.

## The Final Test

Read your document aloud. If you wouldn't say it to a colleague, rewrite it. If you can't point to a real example, number, or outcome, you're writing fluff.

Every sentence should survive this question: "Can someone act on this?" If not, cut it or add specifics.

This isn't about perfection. It's about clarity. Write like you've built the thing, broken it, fixed it, and explained it to confused stakeholders at 3 AM. Because that's what we actually do.

