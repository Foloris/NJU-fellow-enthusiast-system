import "dotenv/config";
import { seedDatabase } from "./db.js";

seedDatabase();
console.log("Database initialized.");

