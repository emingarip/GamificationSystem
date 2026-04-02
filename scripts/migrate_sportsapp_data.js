import 'dotenv/config';
import { createClient } from '@supabase/supabase-js';
import neo4j from 'neo4j-driver';

const SUPABASE_URL = process.env.SUPABASE_URL;
const SUPABASE_KEY = process.env.SUPABASE_SERVICE_ROLE_KEY;

// For local testing, force localhost since script runs outside container
const NEO4J_URI = 'bolt://127.0.0.1:7688';
const NEO4J_USER = process.env.NEO4J_USERNAME || 'neo4j';
const NEO4J_PASSWORD = process.env.NEO4J_PASSWORD || 'neo4j_password';

async function migrate() {
    if (!SUPABASE_URL || !SUPABASE_KEY) {
        console.error("Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY in environment");
        process.exit(1);
    }

    const supabase = createClient(SUPABASE_URL, SUPABASE_KEY);
    const driver = neo4j.driver(NEO4J_URI, neo4j.auth.basic(NEO4J_USER, NEO4J_PASSWORD));
    const session = driver.session();

    try {
        console.log("Starting migration from SportsApp Supabase to GamificationSystem Neo4j...");

        // 1. Fetch Users & Balances
        console.log("Fetching users from Supabase...");
        const { data: users, error: userError } = await supabase
            .from('users') 
            .select('id, k_coin_balance, email, username');

        if (userError) throw userError;

        console.log(`Found ${users.length} users. Migrating balances to Neo4j points...`);

        // Migrate Balances to Neo4j User Points
        for (const user of users) {
             const points = user.k_coin_balance || 0;
             const cypher = `
                MERGE (u:User {userId: $userId})
                SET u.points = $points, 
                    u.email = $email,
                    u.name = $username,
                    u.migratedAt = datetime()
             `;
             await session.run(cypher, { 
                 userId: user.id || '', 
                 points, 
                 email: user.email || '', 
                 username: user.username || ('user_' + (user.id || '').substring(0,6))
             });
        }
        
        // 2. Fetch Badges
        console.log("Fetching user badges from Supabase...");
        const { data: badges, error: badgeError } = await supabase
             .from('user_badges')
             .select('user_id, badge_id, unlocked_at, progress');
             
        if (badgeError) throw badgeError;
        
        console.log(`Found ${badges?.length || 0} user-badge records. Migrating to Neo4j...`);

        if (badges && badges.length > 0) {
            for (const badge of badges) {
                 const cypher = `
                    MATCH (u:User {userId: $userId})
                    // Ensure the badge exists before granting
                    MERGE (b:Achievement {badgeId: $badgeId})
                    MERGE (u)-[r:HAS_BADGE]->(b)
                    SET r.earnedAt = $unlockedAt,
                        r.progress = $progress,
                        r.migratedFrom = 'sportsapp'
                 `;
                 await session.run(cypher, {
                     userId: badge.user_id,
                     badgeId: badge.badge_id,
                     unlockedAt: badge.unlocked_at || new Date().toISOString(),
                     progress: badge.progress || 100
                 });
            }
        }

        console.log("Migration completed successfully!");
    } catch (err) {
        console.error("Migration failed:", err);
    } finally {
        await session.close();
        await driver.close();
    }
}

migrate();
