import React from 'react'

export default function CustomActions() {
  return (
    <section className="mx-auto flex min-h-[calc(100vh-18rem)] max-w-4xl items-center">
      <div className="space-y-4 rounded-[1.5rem] border border-black/10 bg-white/90 p-8 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
        <p className="text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-[#244133]">Custom</p>
        <h2 className="text-3xl font-semibold text-[#18202a]">Custom page</h2>
        <p className="max-w-2xl text-base leading-7 text-[#6a7078]">
          Internal custom view available from the footer.
        </p>
      </div>
    </section>
  )
}
