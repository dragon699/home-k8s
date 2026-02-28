import React from 'react'

export default function ApplicationActions() {
  return (
    <section className="mx-auto flex min-h-[calc(100vh-18rem)] max-w-6xl items-center">
      <div className="grid w-full gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
        <div className="space-y-8">
          <div className="inline-flex items-center rounded-full border border-[#d8d1c7] bg-white/80 px-4 py-2 text-[0.72rem] font-semibold uppercase tracking-[0.32em] text-[#c5392a] shadow-[0_10px_30px_rgba(28,22,16,0.06)]">
            Control UI
          </div>
          <div className="space-y-5">
            <h1 className="max-w-3xl text-5xl font-semibold leading-[0.96] tracking-[-0.05em] text-[#18202a] sm:text-6xl lg:text-7xl">
              One control surface for your home infrastructure.
            </h1>
            <p className="max-w-2xl text-lg leading-8 text-[#5f6771] sm:text-xl">
              Switch between operational views with a lighter shell, faster page transitions, and a cleaner navigation model anchored at the bottom.
            </p>
          </div>
        </div>

        <div className="rounded-[2rem] border border-[#ddd6cd] bg-[#f6f2ed] p-6 shadow-[0_24px_60px_rgba(26,20,14,0.08)]">
          <div className="space-y-5">
            <div className="rounded-[1.5rem] border border-white/80 bg-white/90 p-5 shadow-[0_10px_30px_rgba(26,20,14,0.05)]">
              <p className="text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-[#c5392a]">Layout</p>
              <p className="mt-3 text-2xl font-semibold text-[#18202a]">Top header removed.</p>
              <p className="mt-2 text-sm leading-7 text-[#6a7078]">The app now opens directly into content, with the visual weight moved to the page body and bottom navigation.</p>
            </div>
            <div className="rounded-[1.5rem] border border-[#e2dbd2] bg-[#18202a] p-5 text-white shadow-[0_20px_45px_rgba(24,32,42,0.18)]">
              <p className="text-[0.72rem] font-semibold uppercase tracking-[0.28em] text-[#f8b3aa]">Motion</p>
              <p className="mt-3 text-2xl font-semibold">Page transitions are directional.</p>
              <p className="mt-2 text-sm leading-7 text-white/70">Internal screens slide left or right over 500ms depending on the navigation direction.</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
