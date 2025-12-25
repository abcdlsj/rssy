import NextAuth from "next-auth"
import GitHub from "next-auth/providers/github"
import Credentials from "next-auth/providers/credentials"
import { PrismaAdapter } from "@auth/prisma-adapter"
import { prisma } from "./prisma"

const isDevMode = process.env.DEV_MODE === "true"

export const { handlers, auth, signIn, signOut } = NextAuth({
  adapter: PrismaAdapter(prisma),
  session: {
    strategy: isDevMode ? "jwt" : "database",
  },
  providers: [
    ...(isDevMode
      ? [
          Credentials({
            name: "Dev Login",
            credentials: {
              email: { label: "Email", type: "email" },
            },
            async authorize(credentials) {
              if (!credentials?.email) return null
              const email = credentials.email as string

              let user = await prisma.user.findUnique({ where: { email } })
              if (!user) {
                user = await prisma.user.create({
                  data: { email, name: email.split("@")[0] },
                })
              }
              return user
            },
          }),
        ]
      : [
          GitHub({
            clientId: process.env.AUTH_GITHUB_ID,
            clientSecret: process.env.AUTH_GITHUB_SECRET,
          }),
        ]),
  ],
  callbacks: {
    jwt: async ({ token, user }) => {
      if (user) {
        token.sub = user.id
      }
      return token
    },
    session: async ({ session, user, token }) => {
      if (session.user) {
        session.user.id = isDevMode ? (token?.sub as string) : user.id
      }
      return session
    },
  },
})
